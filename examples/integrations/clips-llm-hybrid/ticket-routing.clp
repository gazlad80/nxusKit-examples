;;; CLIPS Expert System Rules for Ticket Routing
;;;
;;; This file defines the deterministic business rules for routing
;;; support tickets based on LLM-extracted classification.
;;;
;;; Usage with nxusKit CLIPS provider:
;;;   let clips = ClipsEnvironment::new()?;
;;;   clips.load_rules("ticket-routing.clp")?;
;;;   clips.assert_fact(&classification)?;
;;;   clips.run()?;
;;;   let routing = clips.get_facts::<RoutingDecision>()?;

;;; ============================================================
;;; Template Definitions
;;; ============================================================

(deftemplate ticket-classification
  "LLM-extracted classification from a support ticket"
  (slot category (type SYMBOL)
        (allowed-symbols security infrastructure application general))
  (slot priority (type SYMBOL)
        (allowed-symbols low medium high critical))
  (slot sentiment (type SYMBOL)
        (allowed-symbols positive neutral negative frustrated))
  (slot has-security-keywords (type SYMBOL) (default no)))

(deftemplate routing-decision
  "Output routing decision from expert system"
  (slot team (type SYMBOL))
  (slot sla-hours (type INTEGER))
  (slot escalation-level (type INTEGER)))

;;; ============================================================
;;; Security Rules (highest priority)
;;; ============================================================

(defrule route-security-ticket
  "Security tickets always go to security team with escalation"
  (ticket-classification (category security))
  =>
  (assert (routing-decision
    (team security)
    (sla-hours 4)
    (escalation-level 2))))

(defrule route-security-keywords
  "Any ticket with security keywords gets elevated"
  (ticket-classification
    (category ?cat&~security)
    (has-security-keywords yes))
  =>
  (assert (routing-decision
    (team security)
    (sla-hours 8)
    (escalation-level 1))))

;;; ============================================================
;;; Infrastructure Rules
;;; ============================================================

(defrule route-infrastructure-critical
  "Critical infrastructure goes to SRE with immediate escalation"
  (ticket-classification
    (category infrastructure)
    (priority critical))
  =>
  (assert (routing-decision
    (team sre)
    (sla-hours 2)
    (escalation-level 1))))

(defrule route-infrastructure-high
  "High priority infrastructure goes to SRE"
  (ticket-classification
    (category infrastructure)
    (priority high))
  =>
  (assert (routing-decision
    (team sre)
    (sla-hours 4)
    (escalation-level 1))))

(defrule route-infrastructure-standard
  "Standard infrastructure goes to ops team"
  (ticket-classification
    (category infrastructure)
    (priority ?p&~critical&~high))
  =>
  (assert (routing-decision
    (team operations)
    (sla-hours 24)
    (escalation-level 0))))

;;; ============================================================
;;; Application Rules
;;; ============================================================

(defrule route-application-critical
  "Critical application bugs need immediate developer attention"
  (ticket-classification
    (category application)
    (priority critical))
  =>
  (assert (routing-decision
    (team development)
    (sla-hours 4)
    (escalation-level 1))))

(defrule route-application-high
  "High priority application bugs go to development"
  (ticket-classification
    (category application)
    (priority high))
  =>
  (assert (routing-decision
    (team development)
    (sla-hours 8)
    (escalation-level 0))))

(defrule route-application-standard
  "Standard application issues go to development queue"
  (ticket-classification
    (category application)
    (priority ?p&~critical&~high))
  =>
  (assert (routing-decision
    (team development)
    (sla-hours 24)
    (escalation-level 0))))

;;; ============================================================
;;; General Support Rules
;;; ============================================================

(defrule route-general-frustrated
  "Frustrated customers get priority support"
  (ticket-classification
    (category general)
    (sentiment frustrated))
  =>
  (assert (routing-decision
    (team priority-support)
    (sla-hours 4)
    (escalation-level 0))))

(defrule route-general-default
  "Default routing for general inquiries"
  (ticket-classification
    (category general)
    (sentiment ?s&~frustrated))
  =>
  (assert (routing-decision
    (team general-support)
    (sla-hours 24)
    (escalation-level 0))))

;;; ============================================================
;;; Escalation Rules (run after initial routing)
;;; ============================================================

(defrule escalate-vip-customer
  "VIP customers get automatic escalation (fact asserted by caller)"
  ?rd <- (routing-decision (escalation-level ?level))
  (vip-customer)
  (test (< ?level 1))
  =>
  (modify ?rd (escalation-level 1)))

(defrule reduce-sla-frustrated
  "Reduce SLA for frustrated customers beyond default"
  ?rd <- (routing-decision (sla-hours ?sla))
  (ticket-classification (sentiment frustrated))
  (test (> ?sla 8))
  =>
  (modify ?rd (sla-hours (integer (/ ?sla 2)))))
