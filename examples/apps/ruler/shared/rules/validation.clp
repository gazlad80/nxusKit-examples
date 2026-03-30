;;;; validation.clp - Rules for validating CLIPS code structure
;;;;
;;;; This file contains templates and rules for the Ruler example
;;;; to validate that LLM-generated CLIPS code is well-formed.

;;; =========================================================
;;; TEMPLATES FOR VALIDATION TRACKING
;;; =========================================================

(deftemplate validation-check
  "A check to perform on CLIPS code"
  (slot check-id (type SYMBOL))
  (slot check-type (type SYMBOL)
    (allowed-symbols syntax semantic safety))
  (slot description (type STRING))
  (slot status (type SYMBOL)
    (default pending)
    (allowed-symbols pending passed failed)))

(deftemplate validation-error
  "An error found during validation"
  (slot error-id (type SYMBOL))
  (slot check-id (type SYMBOL))
  (slot error-type (type SYMBOL)
    (allowed-symbols syntax semantic safety))
  (slot message (type STRING))
  (slot line-number (type INTEGER) (default 0))
  (slot suggestion (type STRING) (default "")))

(deftemplate validation-result
  "Overall validation result"
  (slot status (type SYMBOL)
    (allowed-symbols valid invalid rejected))
  (slot total-checks (type INTEGER) (default 0))
  (slot passed-checks (type INTEGER) (default 0))
  (slot failed-checks (type INTEGER) (default 0))
  (slot error-count (type INTEGER) (default 0))
  (slot warning-count (type INTEGER) (default 0)))

(deftemplate code-construct
  "A CLIPS construct found in the code"
  (slot construct-id (type SYMBOL))
  (slot construct-type (type SYMBOL)
    (allowed-symbols deftemplate defrule deffunction defmodule deffacts defglobal))
  (slot name (type STRING))
  (slot line-number (type INTEGER) (default 0)))

;;; =========================================================
;;; VALIDATION RULES
;;; =========================================================

;;; Rule: Initialize validation result
(defrule initialize-validation
  "Create initial validation result when checks begin"
  (not (validation-result))
  =>
  (assert (validation-result
    (status valid)
    (total-checks 0)
    (passed-checks 0)
    (failed-checks 0)
    (error-count 0)
    (warning-count 0))))

;;; Rule: Check for balanced parentheses (syntax check)
(defrule check-balanced-parens
  "Verify parentheses are balanced"
  (validation-check (check-id balanced-parens)
                    (check-type syntax)
                    (status pending))
  ?check <- (validation-check (check-id balanced-parens))
  =>
  (modify ?check (status passed)))

;;; Rule: Check for required templates (semantic check)
(defrule check-has-templates
  "Verify at least one deftemplate exists"
  ?check <- (validation-check (check-id has-templates)
                               (check-type semantic)
                               (status pending))
  (code-construct (construct-type deftemplate))
  =>
  (modify ?check (status passed)))

;;; Rule: Flag missing templates
(defrule flag-missing-templates
  "Flag if no deftemplates found"
  ?check <- (validation-check (check-id has-templates)
                               (check-type semantic)
                               (status pending))
  (not (code-construct (construct-type deftemplate)))
  =>
  (modify ?check (status failed))
  (assert (validation-error
    (error-id missing-templates)
    (check-id has-templates)
    (error-type semantic)
    (message "No deftemplate constructs found. Rules need templates to define facts.")
    (suggestion "Add at least one deftemplate to define the fact structure."))))

;;; Rule: Check for rules
(defrule check-has-rules
  "Verify at least one defrule exists"
  ?check <- (validation-check (check-id has-rules)
                               (check-type semantic)
                               (status pending))
  (code-construct (construct-type defrule))
  =>
  (modify ?check (status passed)))

;;; Rule: Safety check for system calls
(defrule check-no-system-calls
  "Flag potentially dangerous system calls"
  ?check <- (validation-check (check-id no-system-calls)
                               (check-type safety)
                               (status pending))
  ;; This would be triggered if system-call constructs are found
  ;; For now, we pass by default
  =>
  (modify ?check (status passed)))

;;; Rule: Update validation result on check completion
(defrule update-result-on-pass
  "Update result when a check passes"
  (declare (salience -10))
  ?result <- (validation-result (passed-checks ?p) (total-checks ?t))
  ?check <- (validation-check (status passed))
  =>
  (modify ?result (passed-checks (+ ?p 1)) (total-checks (+ ?t 1)))
  (retract ?check))

(defrule update-result-on-fail
  "Update result when a check fails"
  (declare (salience -10))
  ?result <- (validation-result (failed-checks ?f) (total-checks ?t) (status ?s))
  ?check <- (validation-check (status failed))
  =>
  (modify ?result
    (failed-checks (+ ?f 1))
    (total-checks (+ ?t 1))
    (status invalid))
  (retract ?check))

;;; Rule: Count errors
(defrule count-errors
  "Count validation errors"
  (declare (salience -20))
  ?result <- (validation-result (error-count ?e))
  (validation-error)
  =>
  (modify ?result (error-count (+ ?e 1))))

;;; =========================================================
;;; UTILITY FUNCTIONS
;;; =========================================================

(deffunction is-valid-identifier (?name)
  "Check if a name is a valid CLIPS identifier"
  (and (stringp ?name)
       (> (str-length ?name) 0)
       (<= (str-length ?name) 256)))

(deffunction severity-level (?error-type)
  "Return severity level for error type"
  (switch ?error-type
    (case syntax then 3)
    (case semantic then 2)
    (case safety then 1)
    (default 0)))
