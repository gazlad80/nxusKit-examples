;;; Incident Detection Pipeline Stage
;;;
;;; Stage 1 of the Incident Response Pipeline (Branching).
;;; Detects and categorizes incidents from raw events/alerts.
;;;
;;; PIPELINE: incident-detection -> incident-classification -> incident-response-*
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: incident, stage: detection, status: pending)
;;;   - raw-event (event data to analyze)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: completed, route-to: classification)
;;;   - detected-incident (structured incident data)
;;;   - detection-indicator (evidence supporting detection)
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
        (stage-name detection)
        (stage-order 1)
        (accepts-types "incident,event")
        (produces-types "incident")
        (can-route-to "classification")))

;;; ==========================================================================
;;; Domain Templates
;;; ==========================================================================

(deftemplate raw-event
    "Raw event data from monitoring systems"
    (slot event-id (type STRING))
    (slot source (type SYMBOL)
        (allowed-symbols firewall ids network endpoint application user log)
        (default log))
    (slot severity-score (type INTEGER) (range 0 100) (default 50))
    (slot event-type (type STRING))
    (slot source-ip (type STRING) (default ""))
    (slot dest-ip (type STRING) (default ""))
    (slot user-id (type STRING) (default ""))
    (slot resource (type STRING) (default ""))
    (slot action (type STRING) (default ""))
    (slot timestamp (type STRING))
    (slot raw-data (type STRING) (default "")))

(deftemplate detected-incident
    "Incident detected from event analysis"
    (slot incident-id (type STRING))
    (slot event-id (type STRING))
    (slot incident-type (type SYMBOL)
        (allowed-symbols security operations performance compliance unknown))
    (slot category (type SYMBOL)
        (allowed-symbols intrusion malware data-breach policy-violation
                         service-outage degradation access-anomaly unknown))
    (slot confidence (type SYMBOL)
        (allowed-symbols high medium low)
        (default medium))
    (slot initial-severity (type SYMBOL)
        (allowed-symbols critical high medium low info)
        (default medium))
    (slot affected-systems (type STRING) (default ""))
    (slot indicators-count (type INTEGER) (default 0))
    (slot source-stage (type SYMBOL) (default detection)))

(deftemplate detection-indicator
    "Evidence supporting incident detection"
    (slot incident-id (type STRING))
    (slot indicator-type (type SYMBOL))
    (slot description (type STRING))
    (slot weight (type INTEGER) (range 1 10) (default 5)))

;;; ==========================================================================
;;; Detection Rules - Security Incidents
;;; ==========================================================================

;;; Mark item as processing
(defrule start-detection
    "Begin processing a pending detection request"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type incident)
        (stage detection)
        (status pending))
    (raw-event (event-id ?id))
    =>
    (modify ?item (status processing)))

;;; Detect brute force attack
(defrule detect-brute-force
    "Multiple failed login attempts indicate brute force"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event
        (event-id ?id)
        (source ?src&:(or (eq ?src firewall) (eq ?src ids) (eq ?src endpoint)))
        (event-type ?type&:(str-index "login_failed" ?type))
        (severity-score ?sev&:(> ?sev 60))
        (source-ip ?ip&:(neq ?ip "")))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type brute-force-attempt)
        (description "Multiple failed login attempts detected")
        (weight 8)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type security)
        (category intrusion)
        (confidence high)
        (initial-severity high)
        (source-stage detection))))

;;; Detect malware activity
(defrule detect-malware-signature
    "Known malware signature detected"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event
        (event-id ?id)
        (source endpoint)
        (event-type ?type&:(or (str-index "malware" ?type) (str-index "virus" ?type)))
        (severity-score ?sev&:(> ?sev 70)))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type malware-signature)
        (description "Known malware signature match")
        (weight 9)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type security)
        (category malware)
        (confidence high)
        (initial-severity critical)
        (source-stage detection))))

;;; Detect data exfiltration attempt
(defrule detect-data-exfiltration
    "Large data transfer to external destination"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event
        (event-id ?id)
        (source network)
        (event-type ?type&:(str-index "transfer" ?type))
        (severity-score ?sev&:(> ?sev 65))
        (dest-ip ?ip&:(neq ?ip "")))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type data-exfiltration)
        (description "Unusual large data transfer to external IP")
        (weight 8)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type security)
        (category data-breach)
        (confidence medium)
        (initial-severity high)
        (source-stage detection))))

;;; ==========================================================================
;;; Detection Rules - Operations Incidents
;;; ==========================================================================

;;; Detect service outage
(defrule detect-service-outage
    "Service health check failures"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event
        (event-id ?id)
        (source application)
        (event-type ?type&:(or (str-index "health_fail" ?type) (str-index "down" ?type)))
        (resource ?res&:(neq ?res "")))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type service-failure)
        (description "Service health check failure detected")
        (weight 7)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type operations)
        (category service-outage)
        (confidence high)
        (initial-severity high)
        (affected-systems ?res)
        (source-stage detection))))

;;; Detect performance degradation
(defrule detect-performance-degradation
    "Response time or throughput degradation"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event
        (event-id ?id)
        (source application)
        (event-type ?type&:(str-index "slow" ?type))
        (severity-score ?sev&:(and (> ?sev 40) (<= ?sev 70))))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type performance-issue)
        (description "Response time degradation detected")
        (weight 5)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type operations)
        (category degradation)
        (confidence medium)
        (initial-severity medium)
        (source-stage detection))))

;;; ==========================================================================
;;; Detection Rules - Access Anomalies
;;; ==========================================================================

;;; Detect unusual access pattern
(defrule detect-access-anomaly
    "Access from unusual location or time"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event
        (event-id ?id)
        (source ?src&:(or (eq ?src ids) (eq ?src user)))
        (event-type ?type&:(str-index "access" ?type))
        (severity-score ?sev&:(> ?sev 50))
        (user-id ?uid&:(neq ?uid "")))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type access-anomaly)
        (description "Unusual access pattern for user")
        (weight 6)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type security)
        (category access-anomaly)
        (confidence medium)
        (initial-severity medium)
        (source-stage detection))))

;;; ==========================================================================
;;; Fallback & Completion Rules
;;; ==========================================================================

;;; Unknown event type - create low-confidence incident
(defrule detect-unknown-event
    "Create incident for unclassified events"
    (declare (salience 30))
    (pipeline-item (item-id ?id) (status processing))
    (raw-event (event-id ?id) (severity-score ?sev&:(> ?sev 60)))
    (not (detected-incident (event-id ?id)))
    =>
    (assert (detection-indicator
        (incident-id ?id)
        (indicator-type unclassified)
        (description "High severity event with unknown pattern")
        (weight 3)))
    (assert (detected-incident
        (incident-id ?id)
        (event-id ?id)
        (incident-type unknown)
        (category unknown)
        (confidence low)
        (initial-severity medium)
        (source-stage detection))))

;;; Complete detection stage
(defrule complete-detection
    "Finalize detection and route to classification"
    (declare (salience 20))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id))
    =>
    ;; Count indicators
    (bind ?count 0)
    (do-for-all-facts ((?ind detection-indicator)) (eq ?ind:incident-id ?id)
        (bind ?count (+ ?count 1)))
    (modify ?item
        (status completed)
        (route-to classification)
        (source-stage detection)))

;;; No incident detected - skip
(defrule no-incident-detected
    "Low severity events can be skipped"
    (declare (salience 15))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (raw-event (event-id ?id) (severity-score ?sev&:(<= ?sev 40)))
    (not (detected-incident (event-id ?id)))
    =>
    (modify ?item
        (status skipped)
        (source-stage detection)))
