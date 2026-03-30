;;;======================================================
;;; Sudoku Constraint Propagation Rules
;;;
;;; Core elimination rules for Sudoku solving using CLIPS.
;;; These rules implement basic constraint propagation:
;;; - Row elimination
;;; - Column elimination
;;; - Box (3x3) elimination
;;;
;;; All rules have salience 100 (high priority) to ensure
;;; constraint propagation happens before strategy rules.
;;;======================================================

;;; Template for a Sudoku cell
(deftemplate cell
   "A single cell in the Sudoku grid"
   (slot row (type INTEGER) (range 1 9))
   (slot col (type INTEGER) (range 1 9))
   (slot value (type INTEGER) (range 0 9))  ; 0 = unknown
   (multislot candidates (type INTEGER)))    ; possible values 1-9

;;; Template for tracking puzzle state
(deftemplate puzzle-state
   "Current state of the puzzle"
   (slot total-cells (type INTEGER) (default 81))
   (slot solved-cells (type INTEGER) (default 0))
   (slot iterations (type INTEGER) (default 0))
   (slot stuck (type INTEGER) (default 0)))  ; 1 if no progress made

;;; Template for requesting LLM help (hybrid mode)
(deftemplate llm-help-request
   "Request for LLM assistance when stuck"
   (slot row (type INTEGER))
   (slot col (type INTEGER))
   (slot candidates (type STRING))
   (slot context (type STRING)))

;;;======================================================
;;; Initialization Rules
;;;======================================================

;;; Initialize candidates for empty cells
(defrule init-candidates
   "Set initial candidates for cells without values"
   (declare (salience 200))
   ?cell <- (cell (value 0) (candidates $?c&:(= (length$ ?c) 0)))
   =>
   (modify ?cell (candidates 1 2 3 4 5 6 7 8 9)))

;;;======================================================
;;; Row Elimination (salience 100)
;;;======================================================

(defrule eliminate-row
   "Remove a value from candidates in the same row"
   (declare (salience 100))
   (cell (row ?r) (col ?c1) (value ?v&~0))
   ?target <- (cell (row ?r) (col ?c2&~?c1) (value 0) (candidates $?before ?v $?after))
   =>
   (modify ?target (candidates $?before $?after)))

;;;======================================================
;;; Column Elimination (salience 100)
;;;======================================================

(defrule eliminate-column
   "Remove a value from candidates in the same column"
   (declare (salience 100))
   (cell (row ?r1) (col ?c) (value ?v&~0))
   ?target <- (cell (row ?r2&~?r1) (col ?c) (value 0) (candidates $?before ?v $?after))
   =>
   (modify ?target (candidates $?before $?after)))

;;;======================================================
;;; Box Elimination (salience 100)
;;;======================================================

(defrule eliminate-box
   "Remove a value from candidates in the same 3x3 box"
   (declare (salience 100))
   (cell (row ?r1) (col ?c1) (value ?v&~0))
   ?target <- (cell (row ?r2) (col ?c2) (value 0) (candidates $?before ?v $?after))
   (test (and (neq ?r1 ?r2) (neq ?c1 ?c2)))  ; Not same cell
   (test (eq (div (- ?r1 1) 3) (div (- ?r2 1) 3)))  ; Same box row
   (test (eq (div (- ?c1 1) 3) (div (- ?c2 1) 3)))  ; Same box col
   =>
   (modify ?target (candidates $?before $?after)))

;;;======================================================
;;; Naked Single (salience 100)
;;;======================================================

(defrule naked-single
   "When a cell has only one candidate, set its value"
   (declare (salience 100))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates ?v))
   ?state <- (puzzle-state (solved-cells ?s) (iterations ?i))
   =>
   (modify ?cell (value ?v) (candidates))
   (modify ?state (solved-cells (+ ?s 1)) (iterations (+ ?i 1)) (stuck 0)))

;;;======================================================
;;; Helper Functions
;;;======================================================

(deffunction box-id (?row ?col)
   "Calculate the box number (1-9) for a cell"
   (+ (* (div (- ?row 1) 3) 3) (div (- ?col 1) 3) 1))

(deffunction count-solved ()
   "Count the number of solved cells"
   (bind ?count 0)
   (do-for-all-facts ((?c cell)) (neq ?c:value 0)
      (bind ?count (+ ?count 1)))
   ?count)

(deffunction is-solved ()
   "Check if the puzzle is completely solved"
   (eq (count-solved) 81))

(deffunction get-cell-candidates (?row ?col)
   "Get candidates for a specific cell as a string"
   (bind ?result "")
   (do-for-all-facts ((?c cell)) (and (eq ?c:row ?row) (eq ?c:col ?col))
      (bind ?result (implode$ ?c:candidates)))
   ?result)
