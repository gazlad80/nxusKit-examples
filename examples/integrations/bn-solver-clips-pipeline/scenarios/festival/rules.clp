;;; Festival Safety Rules
;;; Validates festival stage assignments against safety constraints
;;;
;;; Pipeline stage 3: Takes solver-optimized band-to-stage assignments
;;; and checks them against safety rules for pyrotechnics, noise, and
;;; overcrowding.

;;; ============================================================
;;; Template Definitions
;;; ============================================================

(deftemplate stage-assignment
    "A band assigned to a stage with predicted crowd"
    (slot stage-id (type INTEGER))
    (slot band-name (type STRING))
    (slot predicted-crowd (type INTEGER))
    (slot has-pyro (type SYMBOL) (allowed-symbols yes no))
    (slot stage-material (type SYMBOL) (allowed-symbols wood metal concrete)))

(deftemplate safety-alert
    "Safety violation or warning"
    (slot alert-type (type STRING))
    (slot severity (type SYMBOL) (allowed-symbols info warning critical))
    (slot stage-id (type INTEGER))
    (slot message (type STRING))
    (slot rule-name (type STRING)))

;;; ============================================================
;;; Critical Safety Rules (salience 95-100)
;;; ============================================================

(defrule no-pyro-on-wood
    "Pyrotechnics forbidden on wooden stages"
    (declare (salience 100))
    (stage-assignment (stage-id ?sid) (has-pyro yes) (stage-material wood) (band-name ?band))
    =>
    (assert (safety-alert
        (alert-type "pyro_violation")
        (severity critical)
        (stage-id ?sid)
        (message (str-cat "Band " ?band " has pyro on wooden stage " ?sid))
        (rule-name "no-pyro-on-wood"))))

(defrule overcrowding-alert
    "Crowd above 4500 triggers evacuation planning"
    (declare (salience 95))
    (stage-assignment (stage-id ?sid) (predicted-crowd ?c&:(> ?c 4500)) (band-name ?band))
    =>
    (assert (safety-alert
        (alert-type "overcrowding")
        (severity critical)
        (stage-id ?sid)
        (message (str-cat "Stage " ?sid " crowd " ?c " near capacity - activate evacuation plan for " ?band))
        (rule-name "overcrowding-alert"))))

;;; ============================================================
;;; Warning Rules (salience 80-90)
;;; ============================================================

(defrule max-decibels-exceeded
    "Crowd above 3000 requires sound limiters"
    (declare (salience 90))
    (stage-assignment (stage-id ?sid) (predicted-crowd ?c&:(> ?c 3000)) (band-name ?band))
    =>
    (assert (safety-alert
        (alert-type "noise_warning")
        (severity warning)
        (stage-id ?sid)
        (message (str-cat "Stage " ?sid " crowd " ?c " exceeds 3000 - sound limiters required for " ?band))
        (rule-name "max-decibels-exceeded"))))
