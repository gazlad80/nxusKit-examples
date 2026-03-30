;;; Incident Response - Escalation Path
;;;
;;; Stage 3C of the Incident Response Pipeline (Escalation Branch).
;;; Handles critical incidents requiring management and cross-team coordination.
;;;
;;; PIPELINE: incident-detection -> incident-classification -> THIS
;;;
;;; INPUT:
;;;   - pipeline-item (item-type: incident, stage: response, status: pending)
;;;   - classified-incident (escalation-required: yes OR response-team: all)
;;;
;;; OUTPUT:
;;;   - pipeline-item (status: completed)
;;;   - response-action (specific actions for all teams)
;;;   - escalation-record (management notifications and approvals)
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
        (stage-name escalation-response)
        (stage-order 3)
        (accepts-types "incident")
        (produces-types "response,escalation")
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
        (allowed-symbols isolate block investigate notify remediate document
                         monitor coordinate approve communicate bridge)
        (default investigate))
    (slot priority (type SYMBOL)
        (allowed-symbols immediate high normal low)
        (default normal))
    (slot assignee (type STRING) (default "incident-commander"))
    (slot description (type STRING))
    (slot automated (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot approval-required (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot completed (type SYMBOL) (allowed-symbols yes no) (default no)))

(deftemplate escalation-record
    "Record of escalation notifications and approvals"
    (slot incident-id (type STRING))
    (slot escalation-level (type INTEGER) (range 1 4) (default 1))
    (slot notified-parties (type STRING))
    (slot bridge-established (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot executive-informed (type SYMBOL) (allowed-symbols yes no) (default no))
    (slot external-parties (type STRING) (default ""))
    (slot source-stage (type SYMBOL) (default escalation-response)))

(deftemplate response-summary
    "Overall response plan summary"
    (slot incident-id (type STRING))
    (slot total-actions (type INTEGER) (default 0))
    (slot immediate-actions (type INTEGER) (default 0))
    (slot estimated-resolution-hours (type INTEGER) (default 24))
    (slot command-structure (type SYMBOL)
        (allowed-symbols standard major-incident crisis)
        (default major-incident))
    (slot source-stage (type SYMBOL) (default escalation-response)))

;;; ==========================================================================
;;; Initialization
;;; ==========================================================================

(defrule start-escalation-response
    "Begin escalation response processing"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type incident)
        (stage response)
        (status pending))
    (classified-incident
        (incident-id ?id)
        (escalation-required yes))
    =>
    (modify ?item (status processing)))

(defrule start-escalation-all-teams
    "Begin escalation for all-team incidents"
    (declare (salience 100))
    ?item <- (pipeline-item
        (item-id ?id)
        (item-type incident)
        (stage response)
        (status pending))
    (classified-incident
        (incident-id ?id)
        (response-team all))
    =>
    (modify ?item (status processing)))

;;; ==========================================================================
;;; Command & Control Rules
;;; ==========================================================================

(defrule establish-command
    "Establish incident command structure"
    (declare (salience 95))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity ?sev))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-command"))
        (action-type coordinate)
        (priority immediate)
        (assignee "incident-commander")
        (description "Activate incident command structure and assign IC role")
        (automated no))))

(defrule establish-bridge
    "Create incident bridge/war room"
    (declare (salience 90))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-bridge"))
        (action-type bridge)
        (priority immediate)
        (description "Open incident bridge call and create war room channel")
        (automated yes))))

;;; ==========================================================================
;;; Notification Rules
;;; ==========================================================================

(defrule notify-management
    "Notify management chain"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity ?sev))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-mgmt-notify"))
        (action-type notify)
        (priority immediate)
        (assignee "on-call-manager")
        (description "Notify VP/Director level management of critical incident")
        (automated yes))))

(defrule notify-executive-critical
    "Executive notification for critical incidents"
    (declare (salience 90))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity critical))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-exec-notify"))
        (action-type notify)
        (priority immediate)
        (assignee "executive-team")
        (description "Notify C-level executives of critical incident")
        (automated yes))))

(defrule notify-legal-data-breach
    "Legal and compliance notification for data breaches"
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
        (description "Engage legal counsel for regulatory and disclosure requirements")
        (automated no)
        (approval-required yes))))

(defrule notify-pr-communications
    "Prepare external communications"
    (declare (salience 80))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity ?sev&:(or (eq ?sev critical) (eq ?sev high))))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-comms"))
        (action-type communicate)
        (priority high)
        (assignee "communications-team")
        (description "Prepare customer and public communications if needed")
        (automated no)
        (approval-required yes))))

;;; ==========================================================================
;;; Cross-Team Coordination Rules
;;; ==========================================================================

(defrule coordinate-security-team
    "Engage security team for escalated security incidents"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (incident-type security))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-sec-coord"))
        (action-type coordinate)
        (priority immediate)
        (assignee "security-team")
        (description "Security team lead to join incident bridge")
        (automated no))))

(defrule coordinate-ops-team
    "Engage operations team for infrastructure issues"
    (declare (salience 85))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (incident-type operations))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-ops-coord"))
        (action-type coordinate)
        (priority immediate)
        (assignee "ops-team")
        (description "Operations team lead to join incident bridge")
        (automated no))))

(defrule coordinate-development-team
    "Engage development if code changes needed"
    (declare (salience 75))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity ?sev&:(or (eq ?sev critical) (eq ?sev high))))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-dev-coord"))
        (action-type coordinate)
        (priority high)
        (assignee "dev-team")
        (description "Development team on standby for emergency fixes")
        (automated no))))

;;; ==========================================================================
;;; Approval Rules
;;; ==========================================================================

(defrule require-approval-containment
    "Major containment actions require approval"
    (declare (salience 70))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (category ?cat&:(or (eq ?cat data-breach) (eq ?cat service-outage))))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-containment-approval"))
        (action-type approve)
        (priority immediate)
        (assignee "incident-commander")
        (description "Approve major containment actions (system isolation, service shutdown)")
        (automated no)
        (approval-required yes))))

;;; ==========================================================================
;;; Escalation Record Creation
;;; ==========================================================================

(defrule create-escalation-record-critical
    "Create escalation record for critical incidents"
    (declare (salience 60))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity critical) (category ?cat))
    =>
    (bind ?external (if (eq ?cat data-breach) then "Regulatory bodies, affected customers" else ""))
    (assert (escalation-record
        (incident-id ?id)
        (escalation-level 3)
        (notified-parties "Security, Operations, Management, Executive, Legal")
        (bridge-established yes)
        (executive-informed yes)
        (external-parties ?external)
        (source-stage escalation-response))))

(defrule create-escalation-record-high
    "Create escalation record for high severity incidents"
    (declare (salience 55))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity high))
    (not (escalation-record (incident-id ?id)))
    =>
    (assert (escalation-record
        (incident-id ?id)
        (escalation-level 2)
        (notified-parties "Security, Operations, Management")
        (bridge-established yes)
        (executive-informed no)
        (source-stage escalation-response))))

;;; ==========================================================================
;;; Common Response Rules
;;; ==========================================================================

(defrule respond-document-timeline
    "Document detailed incident timeline"
    (declare (salience 50))
    (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id))
    =>
    (assert (response-action
        (incident-id ?id)
        (action-id (str-cat ?id "-timeline"))
        (action-type document)
        (priority high)
        (assignee "scribe")
        (description "Maintain real-time incident timeline and action log")
        (automated no))))

;;; ==========================================================================
;;; Completion Rules
;;; ==========================================================================

(defrule complete-escalation-response
    "Finalize escalation response"
    (declare (salience 40))
    ?item <- (pipeline-item (item-id ?id) (status processing))
    (classified-incident (incident-id ?id) (final-severity ?sev))
    (escalation-record (incident-id ?id) (escalation-level ?level))
    =>
    ;; Count actions
    (bind ?total 0)
    (bind ?immediate 0)
    (do-for-all-facts ((?a response-action)) (eq ?a:incident-id ?id)
        (bind ?total (+ ?total 1))
        (if (eq ?a:priority immediate) then (bind ?immediate (+ ?immediate 1))))
    ;; Determine command structure
    (bind ?cmd-struct (if (>= ?level 3) then crisis
                       else (if (= ?level 2) then major-incident else standard)))
    ;; Resolution estimate for escalated incidents
    (bind ?est-hours (if (eq ?sev critical) then 4 else 8))
    (modify ?item (status completed) (source-stage escalation-response))
    (assert (response-summary
        (incident-id ?id)
        (total-actions ?total)
        (immediate-actions ?immediate)
        (estimated-resolution-hours ?est-hours)
        (command-structure ?cmd-struct)
        (source-stage escalation-response))))
