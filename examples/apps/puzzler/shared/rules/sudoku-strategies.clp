;;;======================================================
;;; Sudoku Advanced Strategy Rules
;;;
;;; Additional solving strategies beyond basic elimination:
;;; - Hidden singles (salience 40)
;;; - Stuck detection and LLM help request (salience -100)
;;;
;;; These rules fire after propagation rules (salience 100)
;;; have finished making progress.
;;;======================================================

;;; Import the base templates (assumes sudoku-propagation.clp loaded first)

;;;======================================================
;;; Hidden Single in Row (salience 40)
;;;======================================================

(defrule hidden-single-row
   "If a candidate appears in only one cell in a row, set it"
   (declare (salience 40))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates $? ?v $?))
   (not (cell (row ?r) (col ?c2&~?c) (value 0) (candidates $? ?v $?)))
   (not (cell (row ?r) (value ?v)))  ; Value not already in row
   ?state <- (puzzle-state (solved-cells ?s) (iterations ?i))
   =>
   (modify ?cell (value ?v) (candidates))
   (modify ?state (solved-cells (+ ?s 1)) (iterations (+ ?i 1)) (stuck 0)))

;;;======================================================
;;; Hidden Single in Column (salience 40)
;;;======================================================

(defrule hidden-single-column
   "If a candidate appears in only one cell in a column, set it"
   (declare (salience 40))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates $? ?v $?))
   (not (cell (row ?r2&~?r) (col ?c) (value 0) (candidates $? ?v $?)))
   (not (cell (col ?c) (value ?v)))  ; Value not already in column
   ?state <- (puzzle-state (solved-cells ?s) (iterations ?i))
   =>
   (modify ?cell (value ?v) (candidates))
   (modify ?state (solved-cells (+ ?s 1)) (iterations (+ ?i 1)) (stuck 0)))

;;;======================================================
;;; Hidden Single in Box (salience 40)
;;;======================================================

(defrule hidden-single-box
   "If a candidate appears in only one cell in a box, set it"
   (declare (salience 40))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates $? ?v $?))
   (not (and
      (cell (row ?r2) (col ?c2) (value 0) (candidates $? ?v $?))
      (test (and (neq (+ ?r ?c) (+ ?r2 ?c2))  ; Different cell
                 (eq (div (- ?r 1) 3) (div (- ?r2 1) 3))
                 (eq (div (- ?c 1) 3) (div (- ?c2 1) 3))))))
   ?state <- (puzzle-state (solved-cells ?s) (iterations ?i))
   =>
   (modify ?cell (value ?v) (candidates))
   (modify ?state (solved-cells (+ ?s 1)) (iterations (+ ?i 1)) (stuck 0)))

;;;======================================================
;;; Stuck Detection (salience -50)
;;;======================================================

(defrule detect-stuck
   "Detect when no progress is being made -- unsolved cells remain"
   (declare (salience -50))
   ?state <- (puzzle-state (stuck 0))
   (cell (value 0))  ; At least one unsolved cell
   =>
   (modify ?state (stuck 1)))

;;;======================================================
;;; Trial-and-Error (salience -100)
;;;
;;; When stuck, pick the cell with the fewest candidates
;;; and try the first candidate. If it leads to a
;;; contradiction (empty candidates on any cell), the
;;; puzzle state will stop making progress. This is a
;;; simple one-level guess that handles most Sudoku puzzles.
;;;======================================================

(deftemplate trial
   "Records a trial guess for potential backtracking"
   (slot row (type INTEGER))
   (slot col (type INTEGER))
   (slot guess (type INTEGER)))

(defrule try-guess-2-candidates
   "When stuck, try first candidate of a cell with exactly 2 candidates"
   (declare (salience -100))
   ?state <- (puzzle-state (stuck 1))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates ?first ?second))
   (not (trial (row ?r) (col ?c)))  ; Haven't guessed this cell yet
   =>
   (assert (trial (row ?r) (col ?c) (guess ?first)))
   (modify ?cell (value ?first) (candidates))
   (modify ?state (stuck 0)))

(defrule try-guess-3-candidates
   "When stuck and no 2-candidate cells, try first of 3 candidates"
   (declare (salience -110))
   ?state <- (puzzle-state (stuck 1))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates ?first ?second ?third))
   (not (cell (value 0) (candidates ? ?)))  ; No 2-candidate cells
   (not (trial (row ?r) (col ?c)))
   =>
   (assert (trial (row ?r) (col ?c) (guess ?first)))
   (modify ?cell (value ?first) (candidates))
   (modify ?state (stuck 0)))

(defrule try-guess-many-candidates
   "When stuck and no 2 or 3 candidate cells, try any cell"
   (declare (salience -120))
   ?state <- (puzzle-state (stuck 1))
   ?cell <- (cell (row ?r) (col ?c) (value 0) (candidates ?first $?))
   (not (cell (value 0) (candidates ? ?)))       ; No 2-candidate cells
   (not (cell (value 0) (candidates ? ? ?)))     ; No 3-candidate cells
   (not (trial (row ?r) (col ?c)))
   =>
   (assert (trial (row ?r) (col ?c) (guess ?first)))
   (modify ?cell (value ?first) (candidates))
   (modify ?state (stuck 0)))

;;;======================================================
;;; Puzzle Completion Check
;;;======================================================

(defrule check-complete
   "Check if puzzle is solved"
   (declare (salience -200))
   (not (cell (value 0)))
   ?state <- (puzzle-state)
   =>
   (printout t "Puzzle solved!" crlf)
   (modify ?state (solved-cells 81)))

;;;======================================================
;;; Conflict Detection
;;;======================================================

(defrule detect-conflict
   "Detect when a cell has no remaining candidates (invalid state)"
   (declare (salience 150))
   (cell (row ?r) (col ?c) (value 0) (candidates))
   =>
   (printout t "CONFLICT: Cell (" ?r "," ?c ") has no candidates!" crlf))

;;;======================================================
;;; Output Functions
;;;======================================================

(deffunction print-grid ()
   "Print the current Sudoku grid"
   (loop-for-count (?r 1 9)
      (if (eq (mod (- ?r 1) 3) 0) then
         (printout t "+-------+-------+-------+" crlf))
      (printout t "| ")
      (loop-for-count (?c 1 9)
         (bind ?val 0)
         (do-for-all-facts ((?cell cell))
            (and (eq ?cell:row ?r) (eq ?cell:col ?c))
            (bind ?val ?cell:value))
         (if (eq ?val 0) then
            (printout t ". ")
         else
            (printout t ?val " "))
         (if (eq (mod ?c 3) 0) then
            (printout t "| ")))
      (printout t crlf))
   (printout t "+-------+-------+-------+" crlf))

(deffunction get-solution-json ()
   "Return the solution as a JSON array"
   (bind ?result "[")
   (loop-for-count (?r 1 9)
      (if (> ?r 1) then (bind ?result (str-cat ?result ",")))
      (bind ?result (str-cat ?result "["))
      (loop-for-count (?c 1 9)
         (if (> ?c 1) then (bind ?result (str-cat ?result ",")))
         (bind ?val 0)
         (do-for-all-facts ((?cell cell))
            (and (eq ?cell:row ?r) (eq ?cell:col ?c))
            (bind ?val ?cell:value))
         (bind ?result (str-cat ?result ?val)))
      (bind ?result (str-cat ?result "]")))
   (bind ?result (str-cat ?result "]"))
   ?result)
