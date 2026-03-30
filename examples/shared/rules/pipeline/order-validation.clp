;;; Order Validation Pipeline Stage
;;;
;;; Stage 1 of the Order Processing Pipeline.
;;; Validates incoming orders for completeness and business rules.
;;;
;;; PIPELINE: order-validation -> order-pricing -> order-fulfillment
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: order, stage: validation, status: pending)
;;;   - order-request (order details to validate)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: completed/failed, route-to: pricing/nil)
;;;   - validated-order (if validation passes)
;;;   - validation-issue (if problems found)
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
        (stage-name validation)
        (stage-order 1)
        (accepts-types "order")
        (produces-types "order")
        (can-route-to "pricing,fulfillment")))

;;; ==========================================================================
;;; Domain Templates
;;; ==========================================================================

(deftemplate order-request
    "Incoming order to be validated"
    (slot order-id (type STRING))
    (slot customer-id (type STRING))
    (slot customer-type (type SYMBOL)
        (allowed-symbols retail wholesale internal guest)
        (default retail))
    (slot total-amount (type FLOAT) (default 0.0))
    (slot item-count (type INTEGER) (default 0))
    (slot shipping-address (type STRING) (default ""))
    (slot payment-method (type SYMBOL)
        (allowed-symbols credit debit invoice prepaid crypto)
        (default credit))
    (slot priority (type SYMBOL)
        (allowed-symbols standard express overnight)
        (default standard))
    (slot notes (type STRING) (default "")))

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

(deftemplate validation-issue
    "Issue found during validation"
    (slot order-id (type STRING))
    (slot issue-type (type SYMBOL))
    (slot severity (type SYMBOL) (allowed-symbols warning error blocking))
    (slot field (type STRING))
    (slot message (type STRING)))

;;; ==========================================================================
;;; Validation Rules
;;; ==========================================================================

;;; Mark item as processing
(defrule start-validation
    "Begin processing a pending validation request"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type order)
        (stage validation)
        (status pending))
    (order-request (order-id ?id))
    =>
    (modify ?item (status processing)))

;;; Check for missing customer ID
(defrule validate-customer-required
    "Customer ID is required"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (customer-id ?cid&:(or (eq ?cid "") (eq ?cid nil))))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type missing-field)
        (severity blocking)
        (field "customer-id")
        (message "Customer ID is required"))))

;;; Check for missing shipping address
(defrule validate-shipping-required
    "Shipping address is required"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (shipping-address ?addr&:(eq ?addr "")))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type missing-field)
        (severity blocking)
        (field "shipping-address")
        (message "Shipping address is required"))))

;;; Check minimum order amount
(defrule validate-minimum-amount
    "Order must meet minimum amount"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (total-amount ?amt&:(< ?amt 1.0)) (item-count ?cnt&:(> ?cnt 0)))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type business-rule)
        (severity blocking)
        (field "total-amount")
        (message "Order total must be at least $1.00"))))

;;; Check for empty orders
(defrule validate-has-items
    "Order must have at least one item"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (item-count ?cnt&:(<= ?cnt 0)))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type missing-field)
        (severity blocking)
        (field "item-count")
        (message "Order must contain at least one item"))))

;;; Flag high-value orders for review
(defrule flag-high-value-order
    "Orders over $10,000 require review"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (total-amount ?amt&:(> ?amt 10000.0)))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type review-required)
        (severity warning)
        (field "total-amount")
        (message "High-value order requires review"))))

;;; Flag crypto payments for review
(defrule flag-crypto-payment
    "Crypto payments require additional verification"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (payment-method crypto))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type review-required)
        (severity warning)
        (field "payment-method")
        (message "Cryptocurrency payment requires verification"))))

;;; Flag guest orders
(defrule flag-guest-order
    "Guest orders have limited tracking"
    (declare (salience 60))
    (pipeline-item (item-id ?id) (status processing))
    (order-request (order-id ?id) (customer-type guest))
    =>
    (assert (validation-issue
        (order-id ?id)
        (issue-type informational)
        (severity warning)
        (field "customer-type")
        (message "Guest order - encourage account creation"))))

;;; ==========================================================================
;;; Completion Rules
;;; ==========================================================================

;;; Fail validation if blocking issues exist
(defrule validation-failed
    "Fail the order if blocking issues exist"
    (declare (salience 50))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (validation-issue (order-id ?id) (severity blocking))
    =>
    (modify ?item (status failed) (source-stage validation)))

;;; Complete validation successfully (no blocking issues)
(defrule validation-passed
    "Complete validation if no blocking issues"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (order-request
        (order-id ?id)
        (customer-id ?cust)
        (customer-type ?ctype)
        (total-amount ?amt)
        (item-count ?cnt)
        (shipping-address ?addr)
        (payment-method ?pay)
        (priority ?pri))
    (not (validation-issue (order-id ?id) (severity blocking)))
    =>
    (bind ?needs-review (if (any-factp ((?vi validation-issue))
                                (and (eq ?vi:order-id ?id)
                                     (eq ?vi:severity warning))) then yes else no))
    (bind ?score (if (eq ?needs-review yes) then 80 else 100))
    (modify ?item
        (status completed)
        (route-to pricing)
        (source-stage validation))
    (assert (validated-order
        (order-id ?id)
        (customer-id ?cust)
        (customer-type ?ctype)
        (total-amount ?amt)
        (item-count ?cnt)
        (shipping-address ?addr)
        (payment-method ?pay)
        (priority ?pri)
        (validation-score ?score)
        (requires-review ?needs-review)
        (source-stage validation))))

;;; Skip pricing for prepaid internal orders
(defrule skip-pricing-for-prepaid-internal
    "Internal prepaid orders can skip pricing"
    (declare (salience 45))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (order-request
        (order-id ?id)
        (customer-id ?cust)
        (customer-type internal)
        (total-amount ?amt)
        (item-count ?cnt)
        (shipping-address ?addr)
        (payment-method prepaid)
        (priority ?pri))
    (not (validation-issue (order-id ?id) (severity blocking)))
    =>
    (modify ?item
        (status completed)
        (route-to fulfillment)
        (source-stage validation))
    (assert (validated-order
        (order-id ?id)
        (customer-id ?cust)
        (customer-type internal)
        (total-amount ?amt)
        (item-count ?cnt)
        (shipping-address ?addr)
        (payment-method prepaid)
        (priority ?pri)
        (validation-score 100)
        (requires-review no)
        (source-stage validation))))
