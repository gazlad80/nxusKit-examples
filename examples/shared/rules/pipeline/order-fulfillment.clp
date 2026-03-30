;;; Order Fulfillment Pipeline Stage
;;;
;;; Stage 3 of the Order Processing Pipeline.
;;; Determines fulfillment method, warehouse, and shipping carrier.
;;;
;;; PIPELINE: order-validation -> order-pricing -> order-fulfillment
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: order, stage: fulfillment, status: pending)
;;;   - priced-order (from pricing stage) OR validated-order (if pricing skipped)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: completed)
;;;   - fulfillment-decision (routing, warehouse, carrier)
;;;   - fulfillment-action (specific actions to take)
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
        (stage-name fulfillment)
        (stage-order 3)
        (accepts-types "order")
        (produces-types "fulfillment")
        (can-route-to "")))

;;; ==========================================================================
;;; Domain Templates
;;; ==========================================================================

;;; Input: priced-order from pricing stage
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
    (slot pricing-tier (type SYMBOL))
    (slot source-stage (type SYMBOL) (default pricing)))

;;; Alternative input: validated-order (if pricing was skipped)
(deftemplate validated-order
    "Order that has passed validation"
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

;;; Passthrough from pricing stage
(deftemplate shipping-rate
    "Calculated shipping rate (from pricing stage)"
    (slot order-id (type STRING))
    (slot rate (type FLOAT))
    (slot carrier (type SYMBOL) (default standard))
    (slot reason (type STRING)))

(deftemplate discount-applied
    "Record of a discount applied to an order (from pricing stage)"
    (slot order-id (type STRING))
    (slot discount-type (type SYMBOL))
    (slot discount-percent (type FLOAT) (default 0.0))
    (slot discount-amount (type FLOAT) (default 0.0))
    (slot reason (type STRING)))

;;; Output: fulfillment-decision
(deftemplate fulfillment-decision
    "Final fulfillment routing decision"
    (slot order-id (type STRING))
    (slot customer-id (type STRING))
    (slot fulfillment-type (type SYMBOL)
        (allowed-symbols warehouse dropship digital pickup held)
        (default warehouse))
    (slot warehouse (type SYMBOL)
        (allowed-symbols east-coast west-coast central europe asia none)
        (default central))
    (slot carrier (type SYMBOL)
        (allowed-symbols usps ups fedex dhl local digital none)
        (default ups))
    (slot estimated-days (type INTEGER) (default 5))
    (slot special-handling (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot hold-reason (type STRING) (default ""))
    (slot source-stage (type SYMBOL) (default fulfillment)))

(deftemplate fulfillment-action
    "Specific action to take for fulfillment"
    (slot order-id (type STRING))
    (slot action-type (type SYMBOL))
    (slot description (type STRING))
    (slot priority (type SYMBOL) (allowed-symbols low normal high urgent) (default normal)))

;;; ==========================================================================
;;; Initialization Rules
;;; ==========================================================================

;;; Start processing from priced-order
(defrule start-fulfillment-priced
    "Begin fulfillment for priced orders"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type order)
        (stage fulfillment)
        (status pending))
    (priced-order (order-id ?id))
    =>
    (modify ?item (status processing)))

;;; Start processing from validated-order (pricing skipped)
(defrule start-fulfillment-validated
    "Begin fulfillment for validated orders (pricing skipped)"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type order)
        (stage fulfillment)
        (status pending))
    (validated-order (order-id ?id))
    (not (priced-order (order-id ?id)))
    =>
    (modify ?item (status processing)))

;;; ==========================================================================
;;; Warehouse Selection Rules
;;; ==========================================================================

;;; West coast shipping
(defrule select-west-coast-warehouse
    "Route to west coast for western addresses"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (or (priced-order (order-id ?id) (shipping-address ?addr&:(str-index "CA" ?addr)))
        (priced-order (order-id ?id) (shipping-address ?addr&:(str-index "WA" ?addr)))
        (priced-order (order-id ?id) (shipping-address ?addr&:(str-index "OR" ?addr))))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type warehouse-selected)
        (description "West coast warehouse - closer to destination")
        (priority normal))))

;;; East coast shipping
(defrule select-east-coast-warehouse
    "Route to east coast for eastern addresses"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (or (priced-order (order-id ?id) (shipping-address ?addr&:(str-index "NY" ?addr)))
        (priced-order (order-id ?id) (shipping-address ?addr&:(str-index "FL" ?addr)))
        (priced-order (order-id ?id) (shipping-address ?addr&:(str-index "MA" ?addr))))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type warehouse-selected)
        (description "East coast warehouse - closer to destination")
        (priority normal))))

;;; ==========================================================================
;;; Carrier Selection Rules
;;; ==========================================================================

;;; Overnight requires FedEx
(defrule overnight-requires-fedex
    "Use FedEx for overnight delivery"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (or (priced-order (order-id ?id) (priority overnight))
        (validated-order (order-id ?id) (priority overnight)))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type carrier-selected)
        (description "FedEx overnight service")
        (priority urgent))))

;;; Express uses UPS
(defrule express-uses-ups
    "Use UPS for express delivery"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (or (priced-order (order-id ?id) (priority express))
        (validated-order (order-id ?id) (priority express)))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type carrier-selected)
        (description "UPS express service")
        (priority high))))

;;; ==========================================================================
;;; Special Handling Rules
;;; ==========================================================================

;;; High-value orders need special handling
(defrule high-value-special-handling
    "Orders over $5000 need signature confirmation"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (priced-order (order-id ?id) (final-amount ?amt&:(> ?amt 5000.0)))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type special-handling)
        (description "Signature required on delivery - high value order")
        (priority high))))

;;; VIP customers get priority
(defrule vip-priority-handling
    "VIP tier gets priority fulfillment"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (priced-order (order-id ?id) (pricing-tier premium))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type priority-pick)
        (description "Premium customer - prioritize in pick queue")
        (priority high))))

;;; Hold for invoice payment
(defrule hold-invoice-payment
    "Hold fulfillment pending invoice payment"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (or (priced-order (order-id ?id) (payment-method invoice) (final-amount ?amt&:(> ?amt 1000.0)))
        (validated-order (order-id ?id) (payment-method invoice) (total-amount ?amt&:(> ?amt 1000.0))))
    =>
    (assert (fulfillment-action
        (order-id ?id)
        (action-type payment-hold)
        (description "Hold pending invoice payment confirmation for orders over $1000")
        (priority high))))

;;; ==========================================================================
;;; Completion Rules
;;; ==========================================================================

;;; Complete fulfillment for priced orders
(defrule complete-fulfillment-priced
    "Generate fulfillment decision for priced orders"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (priced-order
        (order-id ?id)
        (customer-id ?cust)
        (priority ?pri)
        (pricing-tier ?tier))
    =>
    (bind ?ftype warehouse)
    (bind ?wh central)
    (bind ?carrier (if (eq ?pri overnight) then fedex
                    else (if (eq ?pri express) then ups else usps)))
    (bind ?days (if (eq ?pri overnight) then 1
                 else (if (eq ?pri express) then 3 else 5)))
    (bind ?special (if (or (eq ?tier premium) (eq ?pri overnight)) then yes else no))
    ;; Check for payment hold
    (bind ?hold-reason "")
    (if (any-factp ((?a fulfillment-action))
            (and (eq ?a:order-id ?id) (eq ?a:action-type payment-hold)))
     then
        (bind ?ftype held)
        (bind ?hold-reason "Pending invoice payment confirmation"))
    (modify ?item (status completed) (source-stage fulfillment))
    (assert (fulfillment-decision
        (order-id ?id)
        (customer-id ?cust)
        (fulfillment-type ?ftype)
        (warehouse ?wh)
        (carrier ?carrier)
        (estimated-days ?days)
        (special-handling ?special)
        (hold-reason ?hold-reason)
        (source-stage fulfillment))))

;;; Complete fulfillment for validated orders (pricing skipped)
(defrule complete-fulfillment-validated
    "Generate fulfillment decision for validated orders (skip pricing)"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (validated-order
        (order-id ?id)
        (customer-id ?cust)
        (customer-type ?ctype)
        (priority ?pri))
    (not (priced-order (order-id ?id)))
    =>
    (bind ?carrier (if (eq ?pri overnight) then fedex
                    else (if (eq ?pri express) then ups else usps)))
    (bind ?days (if (eq ?pri overnight) then 1
                 else (if (eq ?pri express) then 3 else 5)))
    (modify ?item (status completed) (source-stage fulfillment))
    (assert (fulfillment-decision
        (order-id ?id)
        (customer-id ?cust)
        (fulfillment-type warehouse)
        (warehouse central)
        (carrier ?carrier)
        (estimated-days ?days)
        (special-handling no)
        (hold-reason "")
        (source-stage fulfillment))))
