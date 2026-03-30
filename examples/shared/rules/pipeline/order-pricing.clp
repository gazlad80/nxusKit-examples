;;; Order Pricing Pipeline Stage
;;;
;;; Stage 2 of the Order Processing Pipeline.
;;; Applies pricing rules, discounts, and calculates final totals.
;;;
;;; PIPELINE: order-validation -> order-pricing -> order-fulfillment
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: order, stage: pricing, status: pending)
;;;   - validated-order (from validation stage)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: completed, route-to: fulfillment)
;;;   - priced-order (with discounts and final totals)
;;;   - discount-applied (record of discounts)
;;;
;;; USAGE:
;;;   cargo run --example clips_pipeline --features clips

;;; ==========================================================================
;;; Pipeline Templates (copied from common.clp for self-contained loading)
;;; ==========================================================================

(deftemplate pipeline-item
    "Standard envelope for pipeline-compatible facts"
    (slot item-id (type STRING))
    (slot item-type (type SYMBOL))
    (slot stage (type SYMBOL))
    (slot status (type SYMBOL)
        (allowed-symbols pending processing completed failed routed skipped)
        (default pending))
    (slot route-to (type SYMBOL) (default nil))
    (slot created-at (type STRING) (default ""))
    (slot source-stage (type SYMBOL) (default nil)))

(deftemplate stage-info
    "Identifies this rulebase's role in a pipeline"
    (slot stage-name (type SYMBOL))
    (slot stage-order (type INTEGER) (default 0))
    (slot accepts-types (type STRING) (default ""))
    (slot produces-types (type STRING) (default ""))
    (slot can-route-to (type STRING) (default "")))

;;; ==========================================================================
;;; Stage Identity
;;; ==========================================================================

(deffacts stage-identity
    (stage-info
        (stage-name pricing)
        (stage-order 2)
        (accepts-types "order")
        (produces-types "order")
        (can-route-to "fulfillment")))

;;; ==========================================================================
;;; Domain Templates
;;; ==========================================================================

;;; Input: validated-order from previous stage
(deftemplate validated-order
    "Order that has passed validation (input from stage 1)"
    (slot order-id (type STRING))
    (slot customer-id (type STRING))
    (slot customer-type (type SYMBOL))
    (slot total-amount (type FLOAT))
    (slot item-count (type INTEGER))
    (slot shipping-address (type STRING))
    (slot payment-method (type SYMBOL))
    (slot priority (type SYMBOL))
    (slot validation-score (type INTEGER) (default 100))
    (slot requires-review (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot source-stage (type SYMBOL) (default validation)))

;;; Output: priced-order for next stage
(deftemplate priced-order
    "Order with pricing calculations complete"
    (slot order-id (type STRING))
    (slot customer-id (type STRING))
    (slot customer-type (type SYMBOL))
    (slot original-amount (type FLOAT))
    (slot discount-amount (type FLOAT) (default 0.0))
    (slot shipping-cost (type FLOAT) (default 0.0))
    (slot tax-amount (type FLOAT) (default 0.0))
    (slot final-amount (type FLOAT))
    (slot item-count (type INTEGER))
    (slot shipping-address (type STRING))
    (slot payment-method (type SYMBOL))
    (slot priority (type SYMBOL))
    (slot pricing-tier (type SYMBOL)
        (allowed-symbols standard preferred premium vip)
        (default standard))
    (slot source-stage (type SYMBOL) (default pricing)))

(deftemplate discount-applied
    "Record of a discount applied to an order"
    (slot order-id (type STRING))
    (slot discount-type (type SYMBOL))
    (slot discount-percent (type FLOAT) (default 0.0))
    (slot discount-amount (type FLOAT) (default 0.0))
    (slot reason (type STRING)))

(deftemplate shipping-rate
    "Calculated shipping rate"
    (slot order-id (type STRING))
    (slot rate (type FLOAT))
    (slot carrier (type SYMBOL) (default standard))
    (slot reason (type STRING)))

;;; ==========================================================================
;;; Pricing Rules
;;; ==========================================================================

;;; Mark item as processing
(defrule start-pricing
    "Begin processing a pending pricing request"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type order)
        (stage pricing)
        (status pending))
    (validated-order (order-id ?id))
    =>
    (modify ?item (status processing)))

;;; Wholesale discount
(defrule apply-wholesale-discount
    "Wholesale customers get 15% discount"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (customer-type wholesale) (total-amount ?amt))
    =>
    (assert (discount-applied
        (order-id ?id)
        (discount-type wholesale)
        (discount-percent 15.0)
        (discount-amount (* ?amt 0.15))
        (reason "Wholesale customer discount"))))

;;; Volume discount
(defrule apply-volume-discount
    "Orders with 10+ items get 5% discount"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (item-count ?cnt&:(>= ?cnt 10)) (total-amount ?amt))
    (not (discount-applied (order-id ?id) (discount-type volume)))
    =>
    (assert (discount-applied
        (order-id ?id)
        (discount-type volume)
        (discount-percent 5.0)
        (discount-amount (* ?amt 0.05))
        (reason "Volume discount for 10+ items"))))

;;; Large order discount
(defrule apply-large-order-discount
    "Orders over $5000 get additional 3% discount"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (total-amount ?amt&:(> ?amt 5000.0)))
    (not (discount-applied (order-id ?id) (discount-type large-order)))
    =>
    (assert (discount-applied
        (order-id ?id)
        (discount-type large-order)
        (discount-percent 3.0)
        (discount-amount (* ?amt 0.03))
        (reason "Large order discount (over $5000)"))))

;;; Prepaid discount
(defrule apply-prepaid-discount
    "Prepaid orders get 2% discount"
    (declare (salience 65))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (payment-method prepaid) (total-amount ?amt))
    =>
    (assert (discount-applied
        (order-id ?id)
        (discount-type prepaid)
        (discount-percent 2.0)
        (discount-amount (* ?amt 0.02))
        (reason "Prepaid payment discount"))))

;;; ==========================================================================
;;; Shipping Rules
;;; ==========================================================================

;;; Free shipping for large orders
(defrule free-shipping-large-order
    "Free shipping for orders over $100"
    (declare (salience 60))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (total-amount ?amt&:(>= ?amt 100.0)) (priority standard))
    (not (shipping-rate (order-id ?id)))
    =>
    (assert (shipping-rate
        (order-id ?id)
        (rate 0.0)
        (carrier standard)
        (reason "Free standard shipping on orders $100+"))))

;;; Standard shipping
(defrule standard-shipping
    "Standard shipping rate"
    (declare (salience 55))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (priority standard))
    (not (shipping-rate (order-id ?id)))
    =>
    (assert (shipping-rate
        (order-id ?id)
        (rate 9.99)
        (carrier standard)
        (reason "Standard shipping"))))

;;; Express shipping
(defrule express-shipping
    "Express shipping rate"
    (declare (salience 55))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (priority express))
    =>
    (assert (shipping-rate
        (order-id ?id)
        (rate 19.99)
        (carrier express)
        (reason "Express shipping (2-3 days)"))))

;;; Overnight shipping
(defrule overnight-shipping
    "Overnight shipping rate"
    (declare (salience 55))
    (pipeline-item (item-id ?id) (status processing))
    (validated-order (order-id ?id) (priority overnight))
    =>
    (assert (shipping-rate
        (order-id ?id)
        (rate 39.99)
        (carrier overnight)
        (reason "Overnight shipping (next business day)"))))

;;; ==========================================================================
;;; Final Calculation & Completion
;;; ==========================================================================

;;; Calculate final pricing
(defrule calculate-final-price
    "Calculate final order total"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (validated-order
        (order-id ?id)
        (customer-id ?cust)
        (customer-type ?ctype)
        (total-amount ?base-amt)
        (item-count ?cnt)
        (shipping-address ?addr)
        (payment-method ?pay)
        (priority ?pri))
    (shipping-rate (order-id ?id) (rate ?ship))
    =>
    ;; Sum all discounts
    (bind ?total-discount 0.0)
    (do-for-all-facts ((?d discount-applied)) (eq ?d:order-id ?id)
        (bind ?total-discount (+ ?total-discount ?d:discount-amount)))
    ;; Calculate tax (assume 8% on discounted amount)
    (bind ?taxable (- ?base-amt ?total-discount))
    (bind ?tax (* ?taxable 0.08))
    ;; Final amount
    (bind ?final (+ ?taxable ?tax ?ship))
    ;; Determine pricing tier
    (bind ?tier (if (eq ?ctype wholesale) then premium
                 else (if (> ?base-amt 5000) then preferred
                       else standard)))
    (modify ?item
        (status completed)
        (route-to fulfillment)
        (source-stage pricing))
    (assert (priced-order
        (order-id ?id)
        (customer-id ?cust)
        (customer-type ?ctype)
        (original-amount ?base-amt)
        (discount-amount ?total-discount)
        (shipping-cost ?ship)
        (tax-amount ?tax)
        (final-amount ?final)
        (item-count ?cnt)
        (shipping-address ?addr)
        (payment-method ?pay)
        (priority ?pri)
        (pricing-tier ?tier)
        (source-stage pricing))))
