;;; Incident Response - Operations Team
;;;
;;; Stage 3B of the Incident Response Pipeline (Operations Branch).
;;; Generates response actions for operations/infrastructure incidents.
;;;
;;; PIPELINE: incident-detection -> incident-classification -> THIS
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: incident, stage: response, status: pending)
;;;   - classified-incident (response-team: operations)
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
        (stage-name ops-response)
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
        (allowed-symbols restart failover scale diagnose notify remediate document monitor)
        (default diagnose))
    (slot priority (type SYMBOL)
        (allowed-symbols immediate high normal low)
        (default normal))
    (slot assignee (type STRING) (default "ops-team"))
    (slot description (type STRING))
    (slot automated (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot runbook (type STRING) (default ""))
    (slot completed (type SYMBOL) (allowed-symbols yes no) (default no)))

(deftemplate response-summary
    "Overall response plan summary"
    (slot incident-id (type STRING))
    (slot total-actions (type INTEGER) (default 0))
    (slot immediate-actions (type INTEGER) (default 0))
    (slot estimated-resolution-hours (type INTEGER) (default 24))
    (slot service-status (type SYMBOL)
        (allowed-symbols down degraded recovering normal)
        (default degraded))
    (slot source-stage (type SYMBOL) (default ops-response)))

;;; ==========================================================================
;;; Initialization
;;; ==========================================================================

(defrule start-ops-response
    "Begin operations response processing"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type incident)
        (stage response)
        (status pending))
    (classified-incident (incident-id ?id) (response-team operations))
    =>
    (modify ?item (status processing)))

;;; ==========================================================================
;;; Service Outage Response Rules
;;; ==========================================================================

(defrule respond-outage-failover
    "Initiate failover for service outage"
    (declare (salience 95))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident
        (incident-id ?id)
        (category service-outage)
        (affected-systems ?sys))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-failover"))
        (action-type failover)
        (priority immediate)
        (description (str-cat "Initiate failover to standby for: " ?sys))
        (automated yes)
        (runbook "RB-FAILOVER-001"))))

(defrule respond-outage-notify-stakeholders
    "Notify stakeholders of outage"
    (declare (salience 90))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category service-outage))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-notify"))
        (action-type notify)
        (priority immediate)
        (description "Send outage notification to stakeholders and status page")
        (automated yes)
        (runbook "RB-NOTIFY-001"))))

(defrule respond-outage-diagnose
    "Diagnose root cause of outage"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category service-outage))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-diagnose"))
        (action-type diagnose)
        (priority high)
        (description "Analyze logs, metrics, and traces to identify root cause")
        (automated no)
        (runbook "RB-DIAGNOSE-001"))))

(defrule respond-outage-restart
    "Attempt service restart"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident
        (incident-id ?id)
        (category service-outage)
        (affected-systems ?sys))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-restart"))
        (action-type restart)
        (priority high)
        (description (str-cat "Attempt controlled restart of: " ?sys))
        (automated yes)
        (runbook "RB-RESTART-001"))))

;;; ==========================================================================
;;; Performance Degradation Response Rules
;;; ==========================================================================

(defrule respond-degradation-scale
    "Scale resources for performance issues"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category degradation))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-scale"))
        (action-type scale)
        (priority high)
        (description "Scale up resources (CPU, memory, instances)")
        (automated yes)
        (runbook "RB-SCALE-001"))))

(defrule respond-degradation-diagnose
    "Diagnose performance bottleneck"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category degradation))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-perf-diagnose"))
        (action-type diagnose)
        (priority high)
        (description "Profile application, check database queries, analyze resource usage")
        (automated no)
        (runbook "RB-PERF-DIAGNOSE-001"))))

(defrule respond-degradation-cache-clear
    "Clear caches if needed"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category degradation))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-cache"))
        (action-type remediate)
        (priority normal)
        (description "Clear application and CDN caches if stale data suspected")
        (automated yes)
        (runbook "RB-CACHE-001"))))

;;; ==========================================================================
;;; Unknown Incident Response Rules
;;; ==========================================================================

(defrule respond-unknown-triage
    "Triage unknown incident type"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category unknown))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-triage"))
        (action-type diagnose)
        (priority high)
        (description "Manual triage to determine incident nature and appropriate response")
        (automated no))))

;;; ==========================================================================
;;; Common Response Rules
;;; ==========================================================================

(defrule respond-monitor-enhanced
    "Enable enhanced monitoring"
    (declare (salience 60))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-monitor"))
        (action-type monitor)
        (priority normal)
        (description "Enable enhanced monitoring and alerting thresholds")
        (automated yes)
        (runbook "RB-MONITOR-001"))))

(defrule respond-document-incident
    "Create incident documentation"
    (declare (salience 55))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-document"))
        (action-type document)
        (priority low)
        (description "Document incident timeline, actions taken, and lessons learned")
        (automated no)
        (runbook "RB-POSTMORTEM-001"))))

;;; ==========================================================================
;;; Completion Rules
;;; ==========================================================================

(defrule complete-ops-response
    "Finalize operations response"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category ?cat) (final-severity ?sev))
    =>
    ;; Count actions
    (bind ?total 0)
    (bind ?immediate 0)
    (do-for-all-facts ((?a response-action)) (eq ?a:incident-id ?id)
        (bind ?total (+ ?total 1))
        (if (eq ?a:priority immediate) then (bind ?immediate (+ ?immediate 1))))
    ;; Estimate resolution time
    (bind ?est-hours (if (eq ?cat service-outage) then 2
                      else (if (eq ?sev high) then 4
                            else (if (eq ?sev medium) then 8 else 24))))
    ;; Service status
    (bind ?status (if (eq ?cat service-outage) then down else degraded))
    (modify ?item (status completed) (source-stage ops-response))
    (assert (response-summary
        (incident-id ?id)
        (total-actions ?total)
        (immediate-actions ?immediate)
        (estimated-resolution-hours ?est-hours)
        (service-status ?status)
        (source-stage ops-response))))
