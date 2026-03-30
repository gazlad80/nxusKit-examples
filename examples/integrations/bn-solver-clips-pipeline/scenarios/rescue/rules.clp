;;; Rescue Safety Protocols
;;; Validates rescue team assignments against safety rules
;;;
;;; Pipeline stage 3: Takes solver-optimized team-to-zone assignments
;;; and checks them against operational safety protocols for weather,
;;; terrain, and equipment limitations.

;;; ============================================================
;;; Template Definitions
;;; ============================================================

(deftemplate rescue-assignment
    "A team assigned to a rescue zone"
    (slot team-id (type INTEGER))
    (slot zone-id (type INTEGER))
    (slot team-type (type SYMBOL) (allowed-symbols ground helicopter drone))
    (slot wind-speed (type INTEGER))
    (slot zone-terrain (type SYMBOL) (allowed-symbols urban rural mountainous)))

(deftemplate protocol-alert
    "Safety protocol violation or advisory"
    (slot alert-type (type STRING))
    (slot severity (type SYMBOL) (allowed-symbols info warning critical))
    (slot team-id (type INTEGER))
    (slot zone-id (type INTEGER))
    (slot message (type STRING))
    (slot rule-name (type STRING)))

;;; ============================================================
;;; Critical Safety Rules (salience 100)
;;; ============================================================

(defrule no-helicopter-in-high-wind
    "Helicopters cannot operate in wind >40mph"
    (declare (salience 100))
    (rescue-assignment (team-id ?tid) (zone-id ?zid) (team-type helicopter) (wind-speed ?ws&:(> ?ws 40)))
    =>
    (assert (protocol-alert
        (alert-type "helicopter_grounded")
        (severity critical)
        (team-id ?tid)
        (zone-id ?zid)
        (message (str-cat "Team " ?tid " helicopter grounded in zone " ?zid " - wind " ?ws "mph exceeds 40mph limit"))
        (rule-name "no-helicopter-in-high-wind"))))

;;; ============================================================
;;; Warning Rules (salience 80-90)
;;; ============================================================

(defrule minimum-team-size
    "Mountain rescue requires team of at least 3"
    (declare (salience 90))
    (rescue-assignment (team-id ?tid) (zone-id ?zid) (zone-terrain mountainous))
    =>
    (assert (protocol-alert
        (alert-type "team_advisory")
        (severity warning)
        (team-id ?tid)
        (zone-id ?zid)
        (message (str-cat "Team " ?tid " assigned to mountainous zone " ?zid " - minimum 3-person team required"))
        (rule-name "minimum-team-size"))))

(defrule drone-range-limit
    "Drones have limited range in rural areas"
    (declare (salience 80))
    (rescue-assignment (team-id ?tid) (zone-id ?zid) (team-type drone) (zone-terrain rural))
    =>
    (assert (protocol-alert
        (alert-type "range_warning")
        (severity warning)
        (team-id ?tid)
        (zone-id ?zid)
        (message (str-cat "Team " ?tid " drone in rural zone " ?zid " - limited range, maintain line of sight"))
        (rule-name "drone-range-limit"))))
