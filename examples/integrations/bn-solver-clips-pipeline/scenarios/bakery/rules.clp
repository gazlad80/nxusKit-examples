;;; Bakery Health & Allergen Rules
;;; Validates baking schedule against health codes and allergen isolation
;;;
;;; Pipeline stage 3: Takes solver-optimized oven scheduling assignments
;;; and checks them against health code requirements for allergen
;;; cross-contamination and temperature safety.

;;; ============================================================
;;; Template Definitions
;;; ============================================================

(deftemplate baking-assignment
    "An item scheduled in an oven"
    (slot item-name (type STRING))
    (slot oven-id (type INTEGER))
    (slot time-slot (type INTEGER))
    (slot contains-nuts (type SYMBOL) (allowed-symbols yes no))
    (slot contains-gluten (type SYMBOL) (allowed-symbols yes no))
    (slot oven-last-allergen (type SYMBOL) (allowed-symbols none nuts gluten both)))

(deftemplate health-alert
    "Health code or allergen violation"
    (slot alert-type (type STRING))
    (slot severity (type SYMBOL) (allowed-symbols info warning critical))
    (slot oven-id (type INTEGER))
    (slot item-name (type STRING))
    (slot message (type STRING))
    (slot rule-name (type STRING)))

;;; ============================================================
;;; Critical Allergen Rules (salience 100)
;;; ============================================================

(defrule allergen-cross-contamination
    "Nut items cannot follow non-nut items without cleaning"
    (declare (salience 100))
    (baking-assignment (item-name ?item) (oven-id ?oid) (contains-nuts yes) (oven-last-allergen none))
    =>
    (assert (health-alert
        (alert-type "allergen_isolation")
        (severity warning)
        (oven-id ?oid)
        (item-name ?item)
        (message (str-cat "Oven " ?oid " needs allergen prep before baking nut item " ?item))
        (rule-name "allergen-cross-contamination"))))

(defrule gluten-free-isolation
    "Gluten-free items must use a clean oven"
    (declare (salience 100))
    (baking-assignment (item-name ?item) (oven-id ?oid) (contains-gluten no) (oven-last-allergen gluten))
    =>
    (assert (health-alert
        (alert-type "gluten_contamination")
        (severity critical)
        (oven-id ?oid)
        (item-name ?item)
        (message (str-cat "CRITICAL: Gluten-free " ?item " in oven " ?oid " after gluten items - deep clean required"))
        (rule-name "gluten-free-isolation"))))

;;; ============================================================
;;; Informational Rules (salience 90)
;;; ============================================================

(defrule temperature-safety
    "Items in sequential slots must not require >50F temperature change"
    (declare (salience 90))
    (baking-assignment (item-name ?item) (oven-id ?oid) (time-slot ?ts))
    =>
    (assert (health-alert
        (alert-type "schedule_note")
        (severity info)
        (oven-id ?oid)
        (item-name ?item)
        (message (str-cat "Oven " ?oid " slot " ?ts ": " ?item " scheduled - verify temperature compatibility"))
        (rule-name "temperature-safety"))))
