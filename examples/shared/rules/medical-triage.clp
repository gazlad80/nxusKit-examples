;;; Medical Triage Expert System
;;;
;;; Prioritizes patients based on symptoms and vital signs.
;;; Demonstrates thinking/trace mode for auditable medical decisions.
;;;
;;; USAGE:
;;;   cargo run --example clips_medical_triage --features clips
;;;
;;; INPUT: Facts with templates "patient", "vital-signs", "symptom", "medical-history"
;;; OUTPUT: Facts with templates "triage-priority" (levels 1-5), "alert"
;;;
;;; TRIAGE LEVELS:
;;;   1 = Immediate (resuscitation) - life-threatening
;;;   2 = Emergent - potentially life-threatening
;;;   3 = Urgent - serious but stable
;;;   4 = Less urgent - minor conditions
;;;   5 = Non-urgent - routine care
;;;
;;; KEY FEATURE: Use ThinkingMode::Enabled to see which rules fired (audit trail)
;;;
;;; This example uses deftemplate (not COOL/defclass) as required by nxusKit.
;;;
;;; DISCLAIMER: This is an educational example only. Not for actual medical use.

;;; ==========================================================================
;;; Template Definitions
;;; ==========================================================================

(deftemplate patient
    "Patient demographic and identification"
    (slot id (type STRING))
    (slot name (type STRING))
    (slot age (type INTEGER) (range 0 150))
    (slot gender (type SYMBOL) (allowed-symbols male female other unknown) (default unknown)))

(deftemplate vital-signs
    "Patient vital signs"
    (slot patient-id (type STRING))
    (slot temperature (type FLOAT))           ; Celsius
    (slot heart-rate (type INTEGER))          ; BPM
    (slot systolic-bp (type INTEGER))         ; mmHg
    (slot diastolic-bp (type INTEGER))        ; mmHg
    (slot respiratory-rate (type INTEGER))    ; breaths per minute
    (slot oxygen-saturation (type INTEGER))   ; percentage
    (slot consciousness (type SYMBOL)
        (allowed-symbols alert verbal pain unresponsive)
        (default alert)))

(deftemplate symptom
    "Reported or observed symptom"
    (slot patient-id (type STRING))
    (slot type (type SYMBOL))
    (slot severity (type SYMBOL) (allowed-symbols mild moderate severe) (default moderate))
    (slot duration-hours (type INTEGER) (default 0)))

(deftemplate medical-history
    "Relevant medical history"
    (slot patient-id (type STRING))
    (slot condition (type SYMBOL))
    (slot current (type SYMBOL) (allowed-symbols yes no) (default yes)))

(deftemplate triage-priority
    "Triage assessment result"
    (slot patient-id (type STRING))
    (slot level (type INTEGER) (range 1 5))   ; 1=immediate, 5=non-urgent
    (slot category (type SYMBOL))
    (slot reason (type STRING))
    (slot requires-immediate-attention (type SYMBOL) (allowed-symbols yes no)))

(deftemplate alert
    "Critical alert for medical staff"
    (slot patient-id (type STRING))
    (slot alert-type (type SYMBOL))
    (slot message (type STRING))
    (slot urgency (type SYMBOL) (allowed-symbols critical high medium low)))

;;; ==========================================================================
;;; Critical Condition Rules (Priority 1 - Immediate)
;;; ==========================================================================

(defrule cardiac-arrest-indicators
    "Detect potential cardiac arrest"
    (declare (salience 100))
    (vital-signs (patient-id ?id)
                 (heart-rate ?hr&:(or (< ?hr 30) (> ?hr 180)))
                 (consciousness unresponsive))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 1)
        (category resuscitation)
        (reason "Cardiac arrest indicators: abnormal heart rate with unresponsiveness")
        (requires-immediate-attention yes)))
    (assert (alert
        (patient-id ?id)
        (alert-type cardiac-emergency)
        (message "CODE BLUE - Possible cardiac arrest")
        (urgency critical))))

(defrule respiratory-failure
    "Detect respiratory failure"
    (declare (salience 100))
    (vital-signs (patient-id ?id)
                 (oxygen-saturation ?sat&:(< ?sat 85))
                 (respiratory-rate ?rr&:(or (< ?rr 8) (> ?rr 35))))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 1)
        (category resuscitation)
        (reason "Respiratory failure: critically low oxygen with abnormal breathing rate")
        (requires-immediate-attention yes)))
    (assert (alert
        (patient-id ?id)
        (alert-type respiratory-emergency)
        (message "Respiratory failure - immediate intervention required")
        (urgency critical))))

(defrule severe-chest-pain-elderly
    "High-risk chest pain in elderly patients"
    (declare (salience 95))
    (patient (id ?id) (age ?a&:(>= ?a 60)))
    (symptom (patient-id ?id) (type chest-pain) (severity severe))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 1)
        (category emergent)
        (reason "Severe chest pain in patient over 60 - possible MI")
        (requires-immediate-attention yes)))
    (assert (alert
        (patient-id ?id)
        (alert-type cardiac-emergency)
        (message "Elderly patient with severe chest pain - cardiac workup stat")
        (urgency critical))))

;;; ==========================================================================
;;; Emergent Rules (Priority 2)
;;; ==========================================================================

(defrule high-fever-infection
    "High fever indicating serious infection"
    (declare (salience 80))
    (vital-signs (patient-id ?id) (temperature ?t&:(> ?t 39.5)))
    (symptom (patient-id ?id) (type ?s&:(or (eq ?s confusion) (eq ?s stiff-neck))))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 2)
        (category emergent)
        (reason "High fever with neurological symptoms - possible meningitis")
        (requires-immediate-attention yes))))

(defrule chest-pain-cardiac-history
    "Chest pain with cardiac history"
    (declare (salience 80))
    (symptom (patient-id ?id) (type chest-pain))
    (medical-history (patient-id ?id) (condition cardiac-disease) (current yes))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 2)
        (category emergent)
        (reason "Chest pain with known cardiac history")
        (requires-immediate-attention yes))))

(defrule severe-bleeding
    "Active severe bleeding"
    (declare (salience 85))
    (symptom (patient-id ?id) (type bleeding) (severity severe))
    (vital-signs (patient-id ?id) (systolic-bp ?sbp&:(< ?sbp 90)))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 2)
        (category emergent)
        (reason "Severe bleeding with hypotension - hemorrhagic shock risk")
        (requires-immediate-attention yes))))

;;; ==========================================================================
;;; Urgent Rules (Priority 3)
;;; ==========================================================================

(defrule moderate-respiratory-distress
    "Moderate breathing difficulty"
    (declare (salience 60))
    (symptom (patient-id ?id) (type difficulty-breathing) (severity moderate))
    (vital-signs (patient-id ?id)
                 (oxygen-saturation ?sat&:(and (>= ?sat 90) (< ?sat 95))))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 3)
        (category urgent)
        (reason "Moderate respiratory distress with borderline oxygen levels")
        (requires-immediate-attention no))))

(defrule abdominal-pain-fever
    "Abdominal pain with fever"
    (declare (salience 55))
    (symptom (patient-id ?id) (type abdominal-pain) (severity ?sev&:(or (eq ?sev moderate) (eq ?sev severe))))
    (vital-signs (patient-id ?id) (temperature ?t&:(> ?t 38.0)))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 3)
        (category urgent)
        (reason "Abdominal pain with fever - possible appendicitis or infection")
        (requires-immediate-attention no))))

(defrule elderly-fall
    "Elderly patient after fall"
    (declare (salience 50))
    (patient (id ?id) (age ?a&:(>= ?a 65)))
    (symptom (patient-id ?id) (type fall))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 3)
        (category urgent)
        (reason "Elderly patient post-fall - risk of fracture or head injury")
        (requires-immediate-attention no))))

;;; ==========================================================================
;;; Less Urgent Rules (Priority 4)
;;; ==========================================================================

(defrule mild-fever
    "Mild fever without alarming symptoms"
    (declare (salience 30))
    (vital-signs (patient-id ?id)
                 (temperature ?t&:(and (> ?t 37.5) (<= ?t 38.5)))
                 (consciousness alert))
    (not (symptom (patient-id ?id) (severity severe)))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 4)
        (category less-urgent)
        (reason "Mild fever, patient alert, no severe symptoms")
        (requires-immediate-attention no))))

(defrule minor-injury
    "Minor injury without complications"
    (declare (salience 25))
    (symptom (patient-id ?id) (type injury) (severity mild))
    (vital-signs (patient-id ?id)
                 (systolic-bp ?sbp&:(>= ?sbp 100))
                 (heart-rate ?hr&:(and (>= ?hr 60) (<= ?hr 100))))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 4)
        (category less-urgent)
        (reason "Minor injury with stable vital signs")
        (requires-immediate-attention no))))

;;; ==========================================================================
;;; Non-Urgent Rules (Priority 5)
;;; ==========================================================================

(defrule stable-chronic-condition
    "Stable chronic condition follow-up"
    (declare (salience 10))
    (patient (id ?id))
    (medical-history (patient-id ?id) (condition ?c) (current yes))
    (vital-signs (patient-id ?id)
                 (temperature ?t&:(and (>= ?t 36.0) (<= ?t 37.5)))
                 (oxygen-saturation ?sat&:(>= ?sat 95))
                 (consciousness alert))
    (not (symptom (patient-id ?id) (severity severe)))
    (not (symptom (patient-id ?id) (severity moderate)))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 5)
        (category non-urgent)
        (reason "Stable patient with chronic condition, routine follow-up")
        (requires-immediate-attention no))))

(defrule minor-complaint
    "Minor complaint with normal vitals"
    (declare (salience 5))
    (patient (id ?id))
    (symptom (patient-id ?id) (severity mild))
    (vital-signs (patient-id ?id)
                 (temperature ?t&:(and (>= ?t 36.0) (<= ?t 37.5)))
                 (heart-rate ?hr&:(and (>= ?hr 60) (<= ?hr 100)))
                 (oxygen-saturation ?sat&:(>= ?sat 95))
                 (consciousness alert))
    =>
    (assert (triage-priority
        (patient-id ?id)
        (level 5)
        (category non-urgent)
        (reason "Minor complaint with normal vital signs")
        (requires-immediate-attention no))))

;;; ==========================================================================
;;; Alert Generation Rules
;;; ==========================================================================

(defrule diabetic-high-glucose-symptoms
    "Alert for diabetic emergency signs"
    (declare (salience 70))
    (medical-history (patient-id ?id) (condition diabetes))
    (symptom (patient-id ?id) (type confusion))
    (vital-signs (patient-id ?id) (respiratory-rate ?rr&:(> ?rr 20)))
    =>
    (assert (alert
        (patient-id ?id)
        (alert-type metabolic-emergency)
        (message "Diabetic patient with confusion and rapid breathing - check glucose stat")
        (urgency high))))

(defrule pediatric-high-fever
    "Alert for high fever in young children"
    (declare (salience 75))
    (patient (id ?id) (age ?a&:(< ?a 5)))
    (vital-signs (patient-id ?id) (temperature ?t&:(> ?t 39.0)))
    =>
    (assert (alert
        (patient-id ?id)
        (alert-type pediatric-fever)
        (message "High fever in young child - seizure risk, evaluate promptly")
        (urgency high))))
