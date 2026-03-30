;;; Inventory Management Expert System
;;;
;;; Manages stock levels, generates reorder alerts, and applies pricing rules.
;;; A practical business example demonstrating CLIPS for operations automation.
;;;
;;; USAGE:
;;;   cargo run --example clips_inventory --features clips
;;;
;;; INPUT: Facts loaded from data/inventory-scenario.json
;;;   - product: SKU, name, costs, reorder thresholds
;;;   - inventory: current stock levels per warehouse
;;;   - sales-velocity: daily sales rate and trend
;;;   - incoming-shipment: expected deliveries
;;;
;;; OUTPUT:
;;;   - stock-status: out-of-stock, critical, low, adequate, overstocked
;;;   - reorder-alert: recommendations with urgency and quantities
;;;   - pricing-adjustment: dynamic pricing suggestions
;;;   - warehouse-transfer: inter-warehouse balancing
;;;
;;; KEY FEATURE: Loads facts from external JSON file
;;;
;;; This example uses deftemplate (not COOL/defclass) as required by nxusKit.

;;; ==========================================================================
;;; Template Definitions
;;; ==========================================================================

(deftemplate product
    "Product catalog information"
    (slot sku (type STRING))
    (slot name (type STRING))
    (slot category (type SYMBOL))
    (slot unit-cost (type FLOAT))
    (slot unit-price (type FLOAT))
    (slot reorder-point (type INTEGER) (default 10))
    (slot reorder-quantity (type INTEGER) (default 50))
    (slot supplier (type STRING)))

(deftemplate inventory
    "Current inventory levels"
    (slot sku (type STRING))
    (slot warehouse (type STRING) (default "MAIN"))
    (slot quantity-on-hand (type INTEGER))
    (slot quantity-reserved (type INTEGER) (default 0))
    (slot last-updated (type STRING)))

(deftemplate incoming-shipment
    "Expected incoming inventory"
    (slot sku (type STRING))
    (slot quantity (type INTEGER))
    (slot expected-date (type STRING))
    (slot status (type SYMBOL) (allowed-symbols pending in-transit received) (default pending)))

(deftemplate sales-velocity
    "Product sales rate metrics"
    (slot sku (type STRING))
    (slot daily-average (type FLOAT))
    (slot trend (type SYMBOL) (allowed-symbols increasing stable decreasing) (default stable)))

(deftemplate reorder-alert
    "Generated reorder recommendation"
    (slot sku (type STRING))
    (slot product-name (type STRING))
    (slot current-stock (type INTEGER))
    (slot available-stock (type INTEGER))
    (slot reorder-point (type INTEGER))
    (slot recommended-quantity (type INTEGER))
    (slot urgency (type SYMBOL) (allowed-symbols critical high medium low))
    (slot reason (type STRING))
    (slot supplier (type STRING)))

(deftemplate stock-status
    "Computed stock status"
    (slot sku (type STRING))
    (slot status (type SYMBOL) (allowed-symbols out-of-stock critical low adequate overstocked))
    (slot days-of-supply (type FLOAT))
    (slot message (type STRING)))

(deftemplate pricing-adjustment
    "Dynamic pricing recommendation"
    (slot sku (type STRING))
    (slot current-price (type FLOAT))
    (slot recommended-price (type FLOAT))
    (slot adjustment-percent (type FLOAT))
    (slot reason (type STRING)))

(deftemplate warehouse-transfer
    "Inter-warehouse transfer recommendation"
    (slot sku (type STRING))
    (slot from-warehouse (type STRING))
    (slot to-warehouse (type STRING))
    (slot quantity (type INTEGER))
    (slot reason (type STRING)))

;;; ==========================================================================
;;; Stock Status Rules
;;; ==========================================================================

(defrule calculate-out-of-stock
    "Identify out of stock items"
    (declare (salience 100))
    (inventory (sku ?sku) (quantity-on-hand ?qoh&:(= ?qoh 0)))
    (product (sku ?sku) (name ?name))
    =>
    (assert (stock-status
        (sku ?sku)
        (status out-of-stock)
        (days-of-supply 0.0)
        (message (str-cat "CRITICAL: " ?name " is completely out of stock")))))

(defrule calculate-critical-stock
    "Identify critically low stock (below 50% of reorder point)"
    (declare (salience 90))
    (inventory (sku ?sku)
               (quantity-on-hand ?qoh&:(> ?qoh 0))
               (quantity-reserved ?reserved))
    (product (sku ?sku) (name ?name) (reorder-point ?rp))
    (test (< (- ?qoh ?reserved) (/ ?rp 2)))
    =>
    (assert (stock-status
        (sku ?sku)
        (status critical)
        (days-of-supply 0.0)
        (message (str-cat "CRITICAL: " ?name " stock is critically low")))))

(defrule calculate-low-stock
    "Identify low stock (at or below reorder point)"
    (declare (salience 80))
    (inventory (sku ?sku)
               (quantity-on-hand ?qoh)
               (quantity-reserved ?reserved))
    (product (sku ?sku) (name ?name) (reorder-point ?rp))
    (test (and (> (- ?qoh ?reserved) (/ ?rp 2))
               (<= (- ?qoh ?reserved) ?rp)))
    =>
    (assert (stock-status
        (sku ?sku)
        (status low)
        (days-of-supply 0.0)
        (message (str-cat "LOW: " ?name " at or below reorder point")))))

(defrule calculate-adequate-stock
    "Identify adequate stock levels"
    (declare (salience 70))
    (inventory (sku ?sku)
               (quantity-on-hand ?qoh)
               (quantity-reserved ?reserved))
    (product (sku ?sku) (name ?name) (reorder-point ?rp) (reorder-quantity ?rq))
    (test (and (> (- ?qoh ?reserved) ?rp)
               (<= (- ?qoh ?reserved) (+ ?rp (* ?rq 2)))))
    =>
    (assert (stock-status
        (sku ?sku)
        (status adequate)
        (days-of-supply 0.0)
        (message (str-cat "OK: " ?name " stock level is adequate")))))

(defrule calculate-overstocked
    "Identify overstocked items"
    (declare (salience 70))
    (inventory (sku ?sku)
               (quantity-on-hand ?qoh)
               (quantity-reserved ?reserved))
    (product (sku ?sku) (name ?name) (reorder-point ?rp) (reorder-quantity ?rq))
    (test (> (- ?qoh ?reserved) (+ ?rp (* ?rq 2))))
    =>
    (assert (stock-status
        (sku ?sku)
        (status overstocked)
        (days-of-supply 0.0)
        (message (str-cat "OVERSTOCK: " ?name " may be overstocked")))))

;;; ==========================================================================
;;; Reorder Alert Rules
;;; ==========================================================================

(defrule generate-critical-reorder
    "Generate critical reorder for out-of-stock items"
    (declare (salience 50))
    (stock-status (sku ?sku) (status out-of-stock))
    (product (sku ?sku)
             (name ?name)
             (reorder-quantity ?rq)
             (reorder-point ?rp)
             (supplier ?sup))
    (inventory (sku ?sku) (quantity-on-hand ?qoh) (quantity-reserved ?res))
    =>
    (assert (reorder-alert
        (sku ?sku)
        (product-name ?name)
        (current-stock ?qoh)
        (available-stock (- ?qoh ?res))
        (reorder-point ?rp)
        (recommended-quantity (* ?rq 2))
        (urgency critical)
        (reason "Out of stock - double order recommended")
        (supplier ?sup))))

(defrule generate-urgent-reorder
    "Generate urgent reorder for critically low stock"
    (declare (salience 45))
    (stock-status (sku ?sku) (status critical))
    (product (sku ?sku)
             (name ?name)
             (reorder-quantity ?rq)
             (reorder-point ?rp)
             (supplier ?sup))
    (inventory (sku ?sku) (quantity-on-hand ?qoh) (quantity-reserved ?res))
    (not (incoming-shipment (sku ?sku) (status ?s&:(or (eq ?s pending) (eq ?s in-transit)))))
    =>
    (assert (reorder-alert
        (sku ?sku)
        (product-name ?name)
        (current-stock ?qoh)
        (available-stock (- ?qoh ?res))
        (reorder-point ?rp)
        (recommended-quantity ?rq)
        (urgency high)
        (reason "Critical stock level, no incoming shipment")
        (supplier ?sup))))

(defrule generate-normal-reorder
    "Generate normal reorder for low stock"
    (declare (salience 40))
    (stock-status (sku ?sku) (status low))
    (product (sku ?sku)
             (name ?name)
             (reorder-quantity ?rq)
             (reorder-point ?rp)
             (supplier ?sup))
    (inventory (sku ?sku) (quantity-on-hand ?qoh) (quantity-reserved ?res))
    (not (incoming-shipment (sku ?sku) (status ?s&:(or (eq ?s pending) (eq ?s in-transit)))))
    =>
    (assert (reorder-alert
        (sku ?sku)
        (product-name ?name)
        (current-stock ?qoh)
        (available-stock (- ?qoh ?res))
        (reorder-point ?rp)
        (recommended-quantity ?rq)
        (urgency medium)
        (reason "Stock at reorder point")
        (supplier ?sup))))

(defrule suppress-reorder-with-incoming
    "Lower urgency when shipment is incoming"
    (declare (salience 35))
    (stock-status (sku ?sku) (status low))
    (product (sku ?sku)
             (name ?name)
             (reorder-quantity ?rq)
             (reorder-point ?rp)
             (supplier ?sup))
    (inventory (sku ?sku) (quantity-on-hand ?qoh) (quantity-reserved ?res))
    (incoming-shipment (sku ?sku) (status ?s&:(or (eq ?s pending) (eq ?s in-transit))) (quantity ?iq))
    =>
    (assert (reorder-alert
        (sku ?sku)
        (product-name ?name)
        (current-stock ?qoh)
        (available-stock (- ?qoh ?res))
        (reorder-point ?rp)
        (recommended-quantity 0)
        (urgency low)
        (reason (str-cat "Low stock but " ?iq " units incoming"))
        (supplier ?sup))))

;;; ==========================================================================
;;; Velocity-Based Rules
;;; ==========================================================================

(defrule high-velocity-increase-reorder
    "Increase reorder quantity for fast-moving items"
    (declare (salience 30))
    (sales-velocity (sku ?sku) (trend increasing) (daily-average ?avg&:(> ?avg 10)))
    (stock-status (sku ?sku) (status ?stat&:(or (eq ?stat low) (eq ?stat critical))))
    (product (sku ?sku) (name ?name) (reorder-quantity ?rq))
    =>
    (assert (reorder-alert
        (sku ?sku)
        (product-name ?name)
        (current-stock 0)
        (available-stock 0)
        (reorder-point 0)
        (recommended-quantity (integer (* ?rq 1.5)))
        (urgency high)
        (reason "High velocity product with increasing trend - increase order size")
        (supplier ""))))

(defrule slow-mover-pricing
    "Suggest discount for slow-moving overstocked items"
    (declare (salience 25))
    (stock-status (sku ?sku) (status overstocked))
    (sales-velocity (sku ?sku) (trend decreasing) (daily-average ?avg&:(< ?avg 2)))
    (product (sku ?sku) (unit-price ?price))
    =>
    (assert (pricing-adjustment
        (sku ?sku)
        (current-price ?price)
        (recommended-price (* ?price 0.85))
        (adjustment-percent -15.0)
        (reason "Slow-moving overstock - 15% discount recommended"))))

(defrule fast-mover-pricing
    "Suggest price increase for fast-moving items with low stock"
    (declare (salience 25))
    (stock-status (sku ?sku) (status ?stat&:(or (eq ?stat low) (eq ?stat critical))))
    (sales-velocity (sku ?sku) (trend increasing) (daily-average ?avg&:(> ?avg 15)))
    (product (sku ?sku) (unit-price ?price))
    =>
    (assert (pricing-adjustment
        (sku ?sku)
        (current-price ?price)
        (recommended-price (* ?price 1.10))
        (adjustment-percent 10.0)
        (reason "High demand with limited supply - 10% price increase recommended"))))

;;; ==========================================================================
;;; Multi-Warehouse Rules
;;; ==========================================================================

(defrule suggest-warehouse-transfer
    "Suggest transfer from overstocked warehouse to understocked"
    (declare (salience 20))
    (inventory (sku ?sku) (warehouse ?wh1) (quantity-on-hand ?qoh1) (quantity-reserved ?res1))
    (inventory (sku ?sku) (warehouse ?wh2&:(neq ?wh2 ?wh1)) (quantity-on-hand ?qoh2) (quantity-reserved ?res2))
    (product (sku ?sku) (reorder-point ?rp) (reorder-quantity ?rq))
    (test (and (> (- ?qoh1 ?res1) (+ ?rp ?rq))
               (< (- ?qoh2 ?res2) ?rp)))
    =>
    (bind ?transfer-qty (min (integer (/ (- ?qoh1 ?res1 ?rp) 2))
                             (- ?rp (- ?qoh2 ?res2))))
    (assert (warehouse-transfer
        (sku ?sku)
        (from-warehouse ?wh1)
        (to-warehouse ?wh2)
        (quantity ?transfer-qty)
        (reason (str-cat "Balance stock between " ?wh1 " and " ?wh2)))))
