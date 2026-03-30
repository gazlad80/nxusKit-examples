;;; Common Pipeline Templates
;;;
;;; Shared templates used across all pipeline-compatible rulebases.
;;; Include this file in each pipeline stage for envelope compatibility.
;;;
;;; USAGE: Load before domain-specific rules
;;;   (load "pipeline/common.clp")
;;;   (load "pipeline/order-validation.clp")

;;; ==========================================================================
;;; Pipeline Envelope Templates
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

(deftemplate pipeline-error
    "Error information from pipeline processing"
    (slot item-id (type STRING))
    (slot stage (type SYMBOL))
    (slot error-type (type SYMBOL))
    (slot message (type STRING))
    (slot recoverable (type SYMBOL) (allowed-symbols yes no) (default no)))

(deftemplate pipeline-metric
    "Metrics from pipeline stage execution"
    (slot item-id (type STRING))
    (slot stage (type SYMBOL))
    (slot metric-name (type SYMBOL))
    (slot metric-value (type FLOAT))
    (slot unit (type STRING) (default "")))
