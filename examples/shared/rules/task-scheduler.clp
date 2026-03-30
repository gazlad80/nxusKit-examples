;;; Task Scheduler Expert System
;;;
;;; Schedules tasks based on priorities, dependencies, resources, and deadlines.
;;; Demonstrates streaming modes (chunk-per-fact, chunk-per-rule).
;;;
;;; USAGE:
;;;   cargo run --example clips_scheduler --features clips
;;;
;;; INPUT: Facts loaded from data/scheduler-scenario.json
;;;   - task: id, name, priority (1-10), estimated hours, deadline, status
;;;   - dependency: task-id depends-on another task-id
;;;   - resource: available workers with hours and skills
;;;
;;; OUTPUT:
;;;   - task-status-update: pending -> ready/blocked based on dependencies
;;;   - execution-order: computed priority order for scheduling
;;;   - schedule-entry: task-to-resource assignments
;;;   - scheduling-alert: deadline risks, circular dependencies, overloads
;;;
;;; KEY FEATURE: Streaming modes
;;;   - StreamMode::Fact - one chunk per derived fact
;;;   - StreamMode::Rule - one chunk per rule firing
;;;
;;; This example uses deftemplate (not COOL/defclass) as required by nxusKit.

;;; ==========================================================================
;;; Template Definitions
;;; ==========================================================================

(deftemplate task
    "A task to be scheduled"
    (slot id (type STRING))
    (slot name (type STRING))
    (slot priority (type INTEGER) (range 1 10) (default 5))
    (slot estimated-hours (type FLOAT) (default 1.0))
    (slot deadline (type STRING))
    (slot status (type SYMBOL)
        (allowed-symbols pending ready in-progress blocked completed cancelled)
        (default pending))
    (slot assigned-to (type STRING) (default ""))
    (slot category (type SYMBOL) (default general)))

(deftemplate dependency
    "Task dependency relationship"
    (slot task-id (type STRING))
    (slot depends-on (type STRING)))

(deftemplate resource
    "Available resource/worker"
    (slot id (type STRING))
    (slot name (type STRING))
    (slot available-hours (type FLOAT) (default 8.0))
    (slot skills (type STRING) (default ""))
    (slot current-load (type INTEGER) (default 0)))

(deftemplate schedule-entry
    "A scheduled task assignment"
    (slot task-id (type STRING))
    (slot task-name (type STRING))
    (slot resource-id (type STRING))
    (slot resource-name (type STRING))
    (slot order (type INTEGER))
    (slot reason (type STRING)))

(deftemplate task-status-update
    "Task status change notification"
    (slot task-id (type STRING))
    (slot old-status (type SYMBOL))
    (slot new-status (type SYMBOL))
    (slot reason (type STRING)))

(deftemplate scheduling-alert
    "Alert about scheduling issues"
    (slot task-id (type STRING))
    (slot alert-type (type SYMBOL))
    (slot severity (type SYMBOL) (allowed-symbols info warning critical))
    (slot message (type STRING)))

(deftemplate execution-order
    "Computed execution order for a task"
    (slot task-id (type STRING))
    (slot order (type INTEGER))
    (slot effective-priority (type INTEGER))
    (slot rationale (type STRING)))

;;; ==========================================================================
;;; Dependency Resolution Rules
;;; ==========================================================================

(defrule mark-task-ready-no-deps
    "Mark tasks as ready if they have no dependencies"
    (declare (salience 100))
    (task (id ?id) (status pending))
    (not (dependency (task-id ?id)))
    =>
    (assert (task-status-update
        (task-id ?id)
        (old-status pending)
        (new-status ready)
        (reason "No dependencies - task is ready to start"))))

(defrule mark-task-ready-deps-complete
    "Mark tasks as ready when all dependencies are complete"
    (declare (salience 95))
    (task (id ?id) (status pending))
    (dependency (task-id ?id) (depends-on ?dep-id))
    (task (id ?dep-id) (status completed))
    (not (and (dependency (task-id ?id) (depends-on ?other-dep))
              (task (id ?other-dep) (status ?s&:(neq ?s completed)))))
    =>
    (assert (task-status-update
        (task-id ?id)
        (old-status pending)
        (new-status ready)
        (reason "All dependencies completed"))))

(defrule mark-task-blocked
    "Mark tasks as blocked when dependencies are not complete"
    (declare (salience 90))
    (task (id ?id) (status pending))
    (dependency (task-id ?id) (depends-on ?dep-id))
    (task (id ?dep-id) (status ?s&:(and (neq ?s completed) (neq ?s cancelled))))
    =>
    (assert (task-status-update
        (task-id ?id)
        (old-status pending)
        (new-status blocked)
        (reason (str-cat "Blocked by incomplete task: " ?dep-id)))))

;;; ==========================================================================
;;; Priority Calculation Rules
;;; ==========================================================================

(defrule calculate-high-priority-order
    "High priority tasks (8-10) get early execution order"
    (declare (salience 80))
    (task (id ?id) (name ?name) (priority ?p&:(>= ?p 8)) (status ?s&:(or (eq ?s ready) (eq ?s pending))))
    =>
    (assert (execution-order
        (task-id ?id)
        (order 1)
        (effective-priority (+ ?p 10))
        (rationale "High priority task - schedule first"))))

(defrule calculate-medium-priority-order
    "Medium priority tasks (4-7) get normal execution order"
    (declare (salience 75))
    (task (id ?id) (name ?name) (priority ?p&:(and (>= ?p 4) (< ?p 8))) (status ?s&:(or (eq ?s ready) (eq ?s pending))))
    =>
    (assert (execution-order
        (task-id ?id)
        (order 2)
        (effective-priority ?p)
        (rationale "Medium priority task - schedule after high priority"))))

(defrule calculate-low-priority-order
    "Low priority tasks (1-3) get later execution order"
    (declare (salience 70))
    (task (id ?id) (name ?name) (priority ?p&:(< ?p 4)) (status ?s&:(or (eq ?s ready) (eq ?s pending))))
    =>
    (assert (execution-order
        (task-id ?id)
        (order 3)
        (effective-priority ?p)
        (rationale "Low priority task - schedule when resources available"))))

;;; ==========================================================================
;;; Resource Assignment Rules
;;; ==========================================================================

(defrule assign-ready-task-to-available-resource
    "Assign ready tasks to available resources"
    (declare (salience 60))
    (task (id ?tid) (name ?tname) (status ready) (estimated-hours ?hours) (assigned-to ""))
    (resource (id ?rid) (name ?rname) (available-hours ?avail&:(>= ?avail ?hours)) (current-load ?load&:(< ?load 3)))
    (execution-order (task-id ?tid) (order ?ord))
    =>
    (assert (schedule-entry
        (task-id ?tid)
        (task-name ?tname)
        (resource-id ?rid)
        (resource-name ?rname)
        (order ?ord)
        (reason (str-cat "Assigned based on availability (" ?avail "h free)")))))

(defrule warn-overloaded-resource
    "Warn when a resource is overloaded"
    (declare (salience 55))
    (resource (id ?rid) (name ?rname) (current-load ?load&:(>= ?load 3)))
    =>
    (assert (scheduling-alert
        (task-id "N/A")
        (alert-type resource-overload)
        (severity warning)
        (message (str-cat "Resource " ?rname " (" ?rid ") is overloaded with " ?load " tasks")))))

(defrule warn-no-available-resource
    "Warn when no resource is available for a ready task"
    (declare (salience 50))
    (task (id ?tid) (name ?tname) (status ready) (estimated-hours ?hours))
    (not (resource (available-hours ?avail&:(>= ?avail ?hours)) (current-load ?load&:(< ?load 3))))
    =>
    (assert (scheduling-alert
        (task-id ?tid)
        (alert-type no-resource)
        (severity warning)
        (message (str-cat "No available resource for task: " ?tname " (" ?hours " hours needed)")))))

;;; ==========================================================================
;;; Deadline and Urgency Rules
;;; ==========================================================================

(defrule detect-deadline-conflict
    "Detect tasks that may miss their deadline"
    (declare (salience 85))
    (task (id ?id) (name ?name) (deadline ?d&:(neq ?d "")) (status ?s&:(or (eq ?s pending) (eq ?s blocked))))
    =>
    (assert (scheduling-alert
        (task-id ?id)
        (alert-type deadline-risk)
        (severity warning)
        (message (str-cat "Task '" ?name "' has deadline " ?d " but is still " ?s)))))

(defrule boost-priority-for-deadline
    "Boost effective priority for tasks approaching deadline"
    (declare (salience 82))
    (task (id ?id) (name ?name) (priority ?p) (deadline ?d&:(neq ?d "")) (status ready))
    (execution-order (task-id ?id) (order ?ord) (effective-priority ?ep))
    =>
    (assert (execution-order
        (task-id ?id)
        (order 0)
        (effective-priority (+ ?ep 5))
        (rationale (str-cat "Deadline approaching (" ?d ") - boosted priority")))))

;;; ==========================================================================
;;; Dependency Chain Analysis
;;; ==========================================================================

(defrule detect-circular-dependency
    "Detect circular dependencies"
    (declare (salience 110))
    (dependency (task-id ?a) (depends-on ?b))
    (dependency (task-id ?b) (depends-on ?a))
    =>
    (assert (scheduling-alert
        (task-id ?a)
        (alert-type circular-dependency)
        (severity critical)
        (message (str-cat "Circular dependency detected between " ?a " and " ?b)))))

(defrule detect-blocked-chain
    "Detect when a task is blocked by a blocked task"
    (declare (salience 65))
    (task (id ?id1) (status blocked))
    (dependency (task-id ?id1) (depends-on ?id2))
    (task (id ?id2) (status blocked))
    =>
    (assert (scheduling-alert
        (task-id ?id1)
        (alert-type chain-blocked)
        (severity info)
        (message (str-cat "Task " ?id1 " is in a blocked chain - waiting on " ?id2)))))

;;; ==========================================================================
;;; Category-Based Scheduling
;;; ==========================================================================

(defrule prioritize-critical-category
    "Critical category tasks get highest priority"
    (declare (salience 88))
    (task (id ?id) (category critical) (status ?s&:(or (eq ?s ready) (eq ?s pending))))
    =>
    (assert (execution-order
        (task-id ?id)
        (order 0)
        (effective-priority 20)
        (rationale "Critical category - top priority"))))

(defrule batch-similar-categories
    "Suggest batching tasks of the same category"
    (declare (salience 40))
    (task (id ?id1) (category ?cat&:(neq ?cat general)) (status ready))
    (task (id ?id2&:(neq ?id2 ?id1)) (category ?cat) (status ready))
    =>
    (assert (scheduling-alert
        (task-id ?id1)
        (alert-type batching-opportunity)
        (severity info)
        (message (str-cat "Consider batching with task " ?id2 " (same category: " ?cat ")")))))
