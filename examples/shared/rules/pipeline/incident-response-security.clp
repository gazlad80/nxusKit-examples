;;; Incident Response - Security Team
;;;
;;; Stage 3A of the Incident Response Pipeline (Security Branch).
;;; Generates response actions for security incidents.
;;;
;;; PIPELINE: incident-detection -> incident-classification -> THIS
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: incident, stage: response, status: pending)
;;;   - classified-incident (response-team: security)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: completed)
;;;   - response-action (specific actions to take)
;;;   - response-summary (overall response plan)
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
        (stage-name security-response)
        (stage-order 3)
        (accepts-types "incident")
        (produces-types "response")
        (can-route-to "")))

;;; ==========================================================================
;;; Domain Templates
;;; ==========================================================================

(deftemplate classified-incident
    "Fully classified incident ready for response"
    (slot incident-id (type STRING))
    (slot incident-type (type SYMBOL))
    (slot category (type SYMBOL))
    (slot final-severity (type SYMBOL))
    (slot response-team (type SYMBOL))
    (slot priority-score (type INTEGER))
    (slot escalation-required (type SYMBOL))
    (slot sla-hours (type INTEGER))
    (slot affected-systems (type STRING))
    (slot classification-notes (type STRING))
    (slot source-stage (type SYMBOL)))

(deftemplate response-action
    "Specific response action to execute"
    (slot incident-id (type STRING))
    (slot action-id (type STRING))
    (slot action-type (type SYMBOL)
        (allowed-symbols isolate block investigate notify remediate document monitor)
        (default investigate))
    (slot priority (type SYMBOL)
        (allowed-symbols immediate high normal low)
        (default normal))
    (slot assignee (type STRING) (default "security-team"))
    (slot description (type STRING))
    (slot automated (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot completed (type SYMBOL) (allowed-symbols yes no) (default no)))

(deftemplate response-summary
    "Overall response plan summary"
    (slot incident-id (type STRING))
    (slot total-actions (type INTEGER) (default 0))
    (slot immediate-actions (type INTEGER) (default 0))
    (slot estimated-resolution-hours (type INTEGER) (default 24))
    (slot containment-status (type SYMBOL)
        (allowed-symbols pending contained eradicated recovered)
        (default pending))
    (slot source-stage (type SYMBOL) (default security-response)))

;;; ==========================================================================
;;; Initialization
;;; ==========================================================================

(defrule start-security-response
    "Begin security response processing"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type incident)
        (stage response)
        (status pending))
    (classified-incident (incident-id ?id) (response-team ?team&:(or (eq ?team security) (eq ?team all))))
    =>
    (modify ?item (status processing)))

;;; ==========================================================================
;;; Intrusion Response Rules
;;; ==========================================================================

(defrule respond-intrusion-isolate
    "Isolate compromised systems for intrusion"
    (declare (salience 90))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident
        (incident-id ?id)
        (category intrusion)
        (affected-systems ?sys&:(neq ?sys "")))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-isolate"))
        (action-type isolate)
        (priority immediate)
        (description (str-cat "Isolate affected systems: " ?sys))
        (automated yes))))

(defrule respond-intrusion-block
    "Block attacking IP addresses"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category intrusion))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-block"))
        (action-type block)
        (priority immediate)
        (description "Block source IPs at firewall and WAF")
        (automated yes))))

(defrule respond-intrusion-investigate
    "Forensic investigation for intrusion"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category intrusion))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-forensics"))
        (action-type investigate)
        (priority high)
        (description "Collect and preserve forensic evidence, analyze attack vectors")
        (automated no))))

;;; ==========================================================================
;;; Malware Response Rules
;;; ==========================================================================

(defrule respond-malware-isolate
    "Isolate infected endpoints"
    (declare (salience 90))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category malware))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-quarantine"))
        (action-type isolate)
        (priority immediate)
        (description "Quarantine infected endpoints from network")
        (automated yes))))

(defrule respond-malware-scan
    "Run full malware scans"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category malware))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-scan"))
        (action-type remediate)
        (priority high)
        (description "Run full AV/EDR scans on affected and adjacent systems")
        (automated yes))))

(defrule respond-malware-investigate
    "Investigate malware origin"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category malware))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-origin"))
        (action-type investigate)
        (priority high)
        (description "Determine infection vector and timeline")
        (automated no))))

;;; ==========================================================================
;;; Data Breach Response Rules
;;; ==========================================================================

(defrule respond-breach-contain
    "Contain data breach"
    (declare (salience 95))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category data-breach))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-contain"))
        (action-type isolate)
        (priority immediate)
        (description "Stop ongoing data exfiltration, block egress")
        (automated yes))))

(defrule respond-breach-notify-legal
    "Legal notification for data breach"
    (declare (salience 90))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category data-breach))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-legal"))
        (action-type notify)
        (priority immediate)
        (assignee "legal-team")
        (description "Notify legal team for breach assessment and regulatory requirements")
        (automated no))))

(defrule respond-breach-assess
    "Assess breach scope"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category data-breach))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-assess"))
        (action-type investigate)
        (priority immediate)
        (description "Determine data types and volume affected")
        (automated no))))

;;; ==========================================================================
;;; Access Anomaly Response Rules
;;; ==========================================================================

(defrule respond-access-revoke
    "Revoke suspicious access"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category access-anomaly))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-revoke"))
        (action-type block)
        (priority high)
        (description "Revoke or suspend suspicious user credentials")
        (automated yes))))

(defrule respond-access-verify
    "Verify user identity"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category access-anomaly))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-verify"))
        (action-type investigate)
        (priority high)
        (description "Contact user to verify activity legitimacy")
        (automated no))))

;;; ==========================================================================
;;; Common Response Rules
;;; ==========================================================================

(defrule respond-document
    "Document the incident"
    (declare (salience 60))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-document"))
        (action-type document)
        (priority normal)
        (description "Create incident report and timeline")
        (automated no))))

(defrule respond-monitor
    "Enhanced monitoring"
    (declare (salience 55))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-monitor"))
        (action-type monitor)
        (priority normal)
        (description "Enable enhanced monitoring for related activity")
        (automated yes))))

;;; ==========================================================================
;;; Completion Rules
;;; ==========================================================================

(defrule complete-security-response
    "Finalize security response"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity ?sev))
    =>
    ;; Count actions
    (bind ?total 0)
    (bind ?immediate 0)
    (do-for-all-facts ((?a response-action)) (eq ?a:incident-id ?id)
        (bind ?total (+ ?total 1))
        (if (eq ?a:priority immediate) then (bind ?immediate (+ ?immediate 1))))
    ;; Estimate resolution time
    (bind ?est-hours (if (eq ?sev critical) then 4
                      else (if (eq ?sev high) then 8
                            else (if (eq ?sev medium) then 24 else 48))))
    (modify ?item (status completed) (source-stage security-response))
    (assert (response-summary
        (incident-id ?id)
        (total-actions ?total)
        (immediate-actions ?immediate)
        (estimated-resolution-hours ?est-hours)
        (containment-status pending)
        (source-stage security-response))))
