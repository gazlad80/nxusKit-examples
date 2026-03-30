;;;======================================================
;;; Classification Evaluation Rules for Solver Pattern
;;;
;;; Validates LLM classification output against quality criteria:
;;; - Confidence must meet threshold
;;; - Category must be in allowed set
;;; - Reasoning must be present (if required)
;;;======================================================

;;; Template for classification output from LLM
(deftemplate classification-output
   "LLM classification result to evaluate"
   (slot category (type SYMBOL))
   (slot confidence (type FLOAT) (range 0.0 1.0))
   (slot reasoning (type STRING) (default ""))
   (slot raw-response (type STRING) (default "")))

;;; Template for evaluation configuration
(deftemplate eval-config
   "Evaluation criteria"
   (slot confidence-threshold (type FLOAT) (default 0.7))
   (multislot valid-categories (type SYMBOL))
   (slot require-reasoning (type INTEGER) (default 1)))

;;; Template for evaluation result
(deftemplate evaluation-result
   "Result of validation"
   (slot status (type SYMBOL) (allowed-symbols valid invalid retry))
   (slot failure-type (type SYMBOL)
         (allowed-symbols none low_confidence invalid_category missing_reasoning parse_error))
   (slot suggested-adjustment (type STRING) (default ""))
   (slot extracted-confidence (type FLOAT) (default 0.0)))

;;;======================================================
;;; Validation Rules (salience from high to low)
;;;======================================================

;;; Rule 1: Check for low confidence (salience 100)
(defrule check-low-confidence
   "Detect when confidence is below threshold"
   (declare (salience 100))
   (classification-output (confidence ?conf))
   (eval-config (confidence-threshold ?threshold))
   (test (< ?conf ?threshold))
   (not (evaluation-result))
   =>
   (assert (evaluation-result
      (status retry)
      (failure-type low_confidence)
      (suggested-adjustment "increase_temperature")
      (extracted-confidence ?conf))))

;;; Rule 2: Check for invalid category (salience 90)
(defrule check-invalid-category
   "Detect when category is not in allowed set"
   (declare (salience 90))
   (classification-output (category ?cat) (confidence ?conf))
   (eval-config (valid-categories $?categories))
   (test (not (member$ ?cat ?categories)))
   (not (evaluation-result))
   =>
   (assert (evaluation-result
      (status retry)
      (failure-type invalid_category)
      (suggested-adjustment "decrease_temperature")
      (extracted-confidence ?conf))))

;;; Rule 3: Check for missing reasoning (salience 80)
(defrule check-missing-reasoning
   "Detect when reasoning is required but missing"
   (declare (salience 80))
   (classification-output (reasoning ?r) (confidence ?conf))
   (eval-config (require-reasoning 1))
   (test (or (eq ?r "") (eq ?r nil)))
   (not (evaluation-result))
   =>
   (assert (evaluation-result
      (status retry)
      (failure-type missing_reasoning)
      (suggested-adjustment "enable_thinking")
      (extracted-confidence ?conf))))

;;; Rule 4: Valid classification (salience 0 - runs last if no failures)
(defrule valid-classification
   "Accept classification when all criteria pass"
   (declare (salience 0))
   (classification-output (confidence ?conf))
   (not (evaluation-result))
   =>
   (assert (evaluation-result
      (status valid)
      (failure-type none)
      (suggested-adjustment "")
      (extracted-confidence ?conf))))

;;;======================================================
;;; Helper function to convert evaluation result to JSON
;;;======================================================
(deffunction result-to-json ()
   "Convert evaluation result to JSON format"
   (bind ?result (find-fact ((?r evaluation-result)) TRUE))
   (if (neq ?result FALSE) then
      (bind ?fact (nth$ 1 ?result))
      (format nil
         "{\"status\":\"%s\",\"failure_type\":\"%s\",\"suggested_adjustment\":\"%s\",\"confidence\":%f}"
         (fact-slot-value ?fact status)
         (fact-slot-value ?fact failure-type)
         (fact-slot-value ?fact suggested-adjustment)
         (fact-slot-value ?fact extracted-confidence))
   else
      "{\"status\":\"invalid\",\"failure_type\":\"parse_error\",\"suggested_adjustment\":\"\",\"confidence\":0.0}"))
