;;; Incident Classification Pipeline Stage
;;;
;;; Stage 2 of the Incident Response Pipeline (Branching).
;;; Classifies incidents and determines routing to appropriate response team.
;;;
;;; PIPELINE: incident-detection -> incident-classification -> incident-response-*
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: incident, stage: classification, status: pending)
;;;   - detected-incident (from detection stage)
;;;   - detection-indicator (evidence from detection)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: routed, route-to: security/operations/escalation)
;;;   - classified-incident (with final severity and response team)
;;;
;;; BRANCHING: Sets route-to based on incident type and severity:
;;;   - security -> incident-response-security.clp
;;;   - operations -> incident-response-ops.clp
;;;   - escalation -> incident-response-escalation.clp
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
        (stage-name classification)
        (stage-order 2)
        (accepts-types "incident")
        (produces-types "incident")
        (can-route-to "security,operations,escalation")))

;;; ==========================================================================
;;; Domain Templates
;;; ==========================================================================

;;; Input from detection stage
(deftemplate detected-incident
    "Incident detected from event analysis"
    (slot incident-id (type STRING))
    (slot event-id (type STRING))
    (slot incident-type (type SYMBOL))
    (slot category (type SYMBOL))
    (slot confidence (type SYMBOL))
    (slot initial-severity (type SYMBOL))
    (slot affected-systems (type STRING) (default ""))
    (slot indicators-count (type INTEGER) (default 0))
    (slot source-stage (type SYMBOL) (default detection)))

(deftemplate detection-indicator
    "Evidence supporting incident detection"
    (slot incident-id (type STRING))
    (slot indicator-type (type SYMBOL))
    (slot description (type STRING))
    (slot weight (type INTEGER) (range 1 10) (default 5)))

;;; Output: classified-incident
(deftemplate classified-incident
    "Fully classified incident ready for response"
    (slot incident-id (type STRING))
    (slot incident-type (type SYMBOL))
    (slot category (type SYMBOL))
    (slot final-severity (type SYMBOL)
        (allowed-symbols critical high medium low info)
        (default medium))
    (slot response-team (type SYMBOL)
        (allowed-symbols security operations management all none)
        (default security))
    (slot priority-score (type INTEGER) (range 0 100) (default 50))
    (slot escalation-required (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot sla-hours (type INTEGER) (default 24))
    (slot affected-systems (type STRING) (default ""))
    (slot classification-notes (type STRING) (default ""))
    (slot source-stage (type SYMBOL) (default classification)))

(deftemplate classification-factor
    "Factor influencing classification decision"
    (slot incident-id (type STRING))
    (slot factor-type (type SYMBOL))
    (slot impact (type SYMBOL) (allowed-symbols increase decrease neutral))
    (slot description (type STRING)))

;;; ==========================================================================
;;; Initialization
;;; ==========================================================================

(defrule start-classification
    "Begin processing a pending classification request"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type incident)
        (stage classification)
        (status pending))
    (detected-incident (incident-id ?id))
    =>
    (modify ?item (status processing)))

;;; ==========================================================================
;;; Severity Adjustment Rules
;;; ==========================================================================

;;; Multiple indicators increase severity
(defrule multiple-indicators-increase-severity
    "Multiple detection indicators suggest higher severity"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id))
    (detection-indicator (incident-id ?id) (weight ?w1&:(> ?w1 5)))
    (detection-indicator (incident-id ?id) (weight ?w2&:(> ?w2 5)))
    (test (neq ?w1 ?w2))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type multiple-high-indicators)
        (impact increase)
        (description "Multiple high-weight indicators detected"))))

;;; Critical systems affected
(defrule critical-system-affected
    "Incidents affecting critical systems are more severe"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident
        (incident-id ?id)
        (affected-systems ?sys&:(or (str-index "database" ?sys)
                                     (str-index "auth" ?sys)
                                     (str-index "payment" ?sys))))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type critical-system)
        (impact increase)
        (description "Critical system affected - elevated priority"))))

;;; Low confidence decreases urgency
(defrule low-confidence-decrease
    "Low confidence detections are less urgent"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id) (confidence low))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type low-confidence)
        (impact decrease)
        (description "Low confidence detection - may be false positive"))))

;;; ==========================================================================
;;; Response Team Assignment Rules
;;; ==========================================================================

;;; Security team for security incidents
(defrule assign-security-team
    "Security incidents go to security team"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id) (incident-type security))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type team-assignment)
        (impact neutral)
        (description "Assigned to Security team"))))

;;; Operations team for ops incidents
(defrule assign-operations-team
    "Operations incidents go to ops team"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id) (incident-type operations))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type team-assignment)
        (impact neutral)
        (description "Assigned to Operations team"))))

;;; ==========================================================================
;;; Escalation Rules
;;; ==========================================================================

;;; Critical severity requires escalation
(defrule escalation-critical
    "Critical incidents require immediate escalation"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id) (initial-severity critical))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type escalation-required)
        (impact increase)
        (description "Critical severity - management escalation required"))))

;;; Data breach requires escalation
(defrule escalation-data-breach
    "Data breaches always require escalation"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (detected-incident (incident-id ?id) (category data-breach))
    =>
    (assert (classification-factor
        (incident-id ?id)
        (factor-type escalation-required)
        (impact increase)
        (description "Data breach - legal and management notification required"))))

;;; ==========================================================================
;;; Final Classification Rules
;;; ==========================================================================

;;; Classify security incidents
(defrule classify-security-incident
    "Generate classification for security incidents"
    (declare (salience 50))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (detected-incident
        (incident-id ?id)
        (incident-type security)
        (category ?cat)
        (initial-severity ?sev)
        (confidence ?conf)
        (affected-systems ?sys))
    =>
    ;; Calculate priority score
    (bind ?base-score (if (eq ?sev critical) then 90
                       else (if (eq ?sev high) then 70
                             else (if (eq ?sev medium) then 50 else 30))))
    ;; Adjust for confidence
    (bind ?conf-adj (if (eq ?conf high) then 10
                     else (if (eq ?conf low) then -15 else 0)))
    ;; Check for escalation factors
    (bind ?needs-escalation (any-factp ((?f classification-factor))
        (and (eq ?f:incident-id ?id) (eq ?f:factor-type escalation-required))))
    ;; Final severity may be elevated
    (bind ?final-sev (if ?needs-escalation
                      then (if (or (eq ?sev critical) (eq ?sev high)) then critical else high)
                      else ?sev))
    ;; SLA based on severity
    (bind ?sla (if (eq ?final-sev critical) then 1
                else (if (eq ?final-sev high) then 4
                      else (if (eq ?final-sev medium) then 8 else 24))))
    (modify ?item
        (status routed)
        (route-to (if ?needs-escalation then escalation else security))
        (source-stage classification))
    (assert (classified-incident
        (incident-id ?id)
        (incident-type security)
        (category ?cat)
        (final-severity ?final-sev)
        (response-team (if ?needs-escalation then all else security))
        (priority-score (+ ?base-score ?conf-adj))
        (escalation-required (if ?needs-escalation then yes else no))
        (sla-hours ?sla)
        (affected-systems ?sys)
        (classification-notes "Security incident classified")
        (source-stage classification))))

;;; Classify operations incidents
(defrule classify-operations-incident
    "Generate classification for operations incidents"
    (declare (salience 50))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (detected-incident
        (incident-id ?id)
        (incident-type operations)
        (category ?cat)
        (initial-severity ?sev)
        (affected-systems ?sys))
    =>
    (bind ?base-score (if (eq ?sev critical) then 85
                       else (if (eq ?sev high) then 65
                             else (if (eq ?sev medium) then 45 else 25))))
    (bind ?sla (if (or (eq ?sev critical) (eq ?cat service-outage)) then 1
                else (if (eq ?sev high) then 4 else 8)))
    (bind ?needs-escalation (eq ?cat service-outage))
    (modify ?item
        (status routed)
        (route-to (if ?needs-escalation then escalation else operations))
        (source-stage classification))
    (assert (classified-incident
        (incident-id ?id)
        (incident-type operations)
        (category ?cat)
        (final-severity ?sev)
        (response-team (if ?needs-escalation then all else operations))
        (priority-score ?base-score)
        (escalation-required (if ?needs-escalation then yes else no))
        (sla-hours ?sla)
        (affected-systems ?sys)
        (classification-notes "Operations incident classified")
        (source-stage classification))))

;;; Classify unknown incidents
(defrule classify-unknown-incident
    "Generate classification for unknown incident types"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (detected-incident
        (incident-id ?id)
        (incident-type unknown)
        (initial-severity ?sev))
    =>
    (modify ?item
        (status routed)
        (route-to operations)
        (source-stage classification))
    (assert (classified-incident
        (incident-id ?id)
        (incident-type unknown)
        (category unknown)
        (final-severity medium)
        (response-team operations)
        (priority-score 40)
        (escalation-required no)
        (sla-hours 24)
        (classification-notes "Unknown incident type - assigned to ops for triage")
        (source-stage classification))))
