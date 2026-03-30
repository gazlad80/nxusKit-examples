;;;; einstein-riddle.clp -- Candidate-elimination solver for Einstein's Riddle
;;;;
;;;; PUZZLE: Five houses in a row, each with a unique nationality, house color,
;;;; drink, cigarette brand, and pet. Fifteen clues constrain the solution.
;;;; Goal: determine who owns the fish.
;;;;
;;;; APPROACH: "Generate then eliminate" using (possible pos attr val) facts.
;;;; We start with 125 candidate facts (5 positions x 5 attributes x 5 values).
;;;; Rules retract impossible candidates. When only one candidate remains for a
;;;; position-attribute pair (naked single) or an attribute-value pair (hidden
;;;; single), we assert a (resolved) fact and propagate uniqueness.
;;;;
;;;; Pure forward-chaining -- no backtracking needed because Einstein's Riddle
;;;; is fully determined by its 15 clues.
;;;;
;;;; CLIPS SEMANTICS NOTE:
;;;; Variables in (not ...) CEs are universally quantified within the not.
;;;; A variable used in a (not ...) must already be bound by a preceding
;;;; positive pattern for it to refer to a specific value. Therefore, every
;;;; rule below binds ?p from a positive (possible ...) pattern BEFORE
;;;; testing absence with (not (possible ...)).
;;;;
;;;; THE 15 CLUES:
;;;;  1. The Brit lives in the red house.
;;;;  2. The Swede keeps dogs.
;;;;  3. The Dane drinks tea.
;;;;  4. The green house is immediately to the left of the white house.
;;;;  5. The green house owner drinks coffee.
;;;;  6. The Pall Mall smoker keeps birds.
;;;;  7. The owner of the yellow house smokes Dunhill.
;;;;  8. The person in the center house drinks milk.
;;;;  9. The Norwegian lives in the first house.
;;;; 10. The Blend smoker lives next to the cat owner.
;;;; 11. The person who keeps horses lives next to the Dunhill smoker.
;;;; 12. The Blue Master smoker drinks beer.
;;;; 13. The German smokes Prince.
;;;; 14. The Norwegian lives next to the blue house.
;;;; 15. The Blend smoker has a neighbor who drinks water.
;;;;
;;;; CORRECT SOLUTION:
;;;; House 1: Norwegian, yellow,  water,  Dunhill,    cat
;;;; House 2: Dane,      blue,    tea,    Blend,      horse
;;;; House 3: Brit,       red,     milk,   PallMall,   bird
;;;; House 4: German,    green,   coffee, Prince,     fish   <-- German owns fish
;;;; House 5: Swede,     white,   beer,   BlueMaster, dog

;;; =========================================================
;;; TEMPLATES
;;; =========================================================

(deftemplate possible
  "A candidate assignment: value ?val is possible for attribute ?attr at position ?pos"
  (slot pos  (type INTEGER) (range 1 5))
  (slot attr (type SYMBOL)
    (allowed-symbols nationality color drink cigarette pet))
  (slot val  (type SYMBOL)))

(deftemplate resolved
  "A confirmed assignment: position ?pos has value ?val for attribute ?attr"
  (slot pos  (type INTEGER) (range 1 5))
  (slot attr (type SYMBOL)
    (allowed-symbols nationality color drink cigarette pet))
  (slot val  (type SYMBOL)))

(deftemplate house
  "Convenience view -- one fact per house with all resolved attributes.
   Updated incrementally as attributes resolve."
  (slot position    (type INTEGER) (range 1 5))
  (slot nationality (type SYMBOL) (default unknown))
  (slot color       (type SYMBOL) (default unknown))
  (slot drink       (type SYMBOL) (default unknown))
  (slot cigarette   (type SYMBOL) (default unknown))
  (slot pet         (type SYMBOL) (default unknown)))

(deftemplate solution
  "The final answer"
  (slot fish-owner (type SYMBOL) (default unknown)))

;;; =========================================================
;;; INITIAL FACTS -- 125 candidates (5 pos x 5 attr x 5 val)
;;; =========================================================

(deffacts initial-candidates
  "Every value is initially possible in every position for every attribute"

  ;; --- nationality ---
  (possible (pos 1) (attr nationality) (val Brit))
  (possible (pos 1) (attr nationality) (val Swede))
  (possible (pos 1) (attr nationality) (val Dane))
  (possible (pos 1) (attr nationality) (val Norwegian))
  (possible (pos 1) (attr nationality) (val German))
  (possible (pos 2) (attr nationality) (val Brit))
  (possible (pos 2) (attr nationality) (val Swede))
  (possible (pos 2) (attr nationality) (val Dane))
  (possible (pos 2) (attr nationality) (val Norwegian))
  (possible (pos 2) (attr nationality) (val German))
  (possible (pos 3) (attr nationality) (val Brit))
  (possible (pos 3) (attr nationality) (val Swede))
  (possible (pos 3) (attr nationality) (val Dane))
  (possible (pos 3) (attr nationality) (val Norwegian))
  (possible (pos 3) (attr nationality) (val German))
  (possible (pos 4) (attr nationality) (val Brit))
  (possible (pos 4) (attr nationality) (val Swede))
  (possible (pos 4) (attr nationality) (val Dane))
  (possible (pos 4) (attr nationality) (val Norwegian))
  (possible (pos 4) (attr nationality) (val German))
  (possible (pos 5) (attr nationality) (val Brit))
  (possible (pos 5) (attr nationality) (val Swede))
  (possible (pos 5) (attr nationality) (val Dane))
  (possible (pos 5) (attr nationality) (val Norwegian))
  (possible (pos 5) (attr nationality) (val German))

  ;; --- color ---
  (possible (pos 1) (attr color) (val red))
  (possible (pos 1) (attr color) (val green))
  (possible (pos 1) (attr color) (val white))
  (possible (pos 1) (attr color) (val yellow))
  (possible (pos 1) (attr color) (val blue))
  (possible (pos 2) (attr color) (val red))
  (possible (pos 2) (attr color) (val green))
  (possible (pos 2) (attr color) (val white))
  (possible (pos 2) (attr color) (val yellow))
  (possible (pos 2) (attr color) (val blue))
  (possible (pos 3) (attr color) (val red))
  (possible (pos 3) (attr color) (val green))
  (possible (pos 3) (attr color) (val white))
  (possible (pos 3) (attr color) (val yellow))
  (possible (pos 3) (attr color) (val blue))
  (possible (pos 4) (attr color) (val red))
  (possible (pos 4) (attr color) (val green))
  (possible (pos 4) (attr color) (val white))
  (possible (pos 4) (attr color) (val yellow))
  (possible (pos 4) (attr color) (val blue))
  (possible (pos 5) (attr color) (val red))
  (possible (pos 5) (attr color) (val green))
  (possible (pos 5) (attr color) (val white))
  (possible (pos 5) (attr color) (val yellow))
  (possible (pos 5) (attr color) (val blue))

  ;; --- drink ---
  (possible (pos 1) (attr drink) (val tea))
  (possible (pos 1) (attr drink) (val coffee))
  (possible (pos 1) (attr drink) (val milk))
  (possible (pos 1) (attr drink) (val beer))
  (possible (pos 1) (attr drink) (val water))
  (possible (pos 2) (attr drink) (val tea))
  (possible (pos 2) (attr drink) (val coffee))
  (possible (pos 2) (attr drink) (val milk))
  (possible (pos 2) (attr drink) (val beer))
  (possible (pos 2) (attr drink) (val water))
  (possible (pos 3) (attr drink) (val tea))
  (possible (pos 3) (attr drink) (val coffee))
  (possible (pos 3) (attr drink) (val milk))
  (possible (pos 3) (attr drink) (val beer))
  (possible (pos 3) (attr drink) (val water))
  (possible (pos 4) (attr drink) (val tea))
  (possible (pos 4) (attr drink) (val coffee))
  (possible (pos 4) (attr drink) (val milk))
  (possible (pos 4) (attr drink) (val beer))
  (possible (pos 4) (attr drink) (val water))
  (possible (pos 5) (attr drink) (val tea))
  (possible (pos 5) (attr drink) (val coffee))
  (possible (pos 5) (attr drink) (val milk))
  (possible (pos 5) (attr drink) (val beer))
  (possible (pos 5) (attr drink) (val water))

  ;; --- cigarette ---
  (possible (pos 1) (attr cigarette) (val PallMall))
  (possible (pos 1) (attr cigarette) (val Dunhill))
  (possible (pos 1) (attr cigarette) (val Blend))
  (possible (pos 1) (attr cigarette) (val BlueMaster))
  (possible (pos 1) (attr cigarette) (val Prince))
  (possible (pos 2) (attr cigarette) (val PallMall))
  (possible (pos 2) (attr cigarette) (val Dunhill))
  (possible (pos 2) (attr cigarette) (val Blend))
  (possible (pos 2) (attr cigarette) (val BlueMaster))
  (possible (pos 2) (attr cigarette) (val Prince))
  (possible (pos 3) (attr cigarette) (val PallMall))
  (possible (pos 3) (attr cigarette) (val Dunhill))
  (possible (pos 3) (attr cigarette) (val Blend))
  (possible (pos 3) (attr cigarette) (val BlueMaster))
  (possible (pos 3) (attr cigarette) (val Prince))
  (possible (pos 4) (attr cigarette) (val PallMall))
  (possible (pos 4) (attr cigarette) (val Dunhill))
  (possible (pos 4) (attr cigarette) (val Blend))
  (possible (pos 4) (attr cigarette) (val BlueMaster))
  (possible (pos 4) (attr cigarette) (val Prince))
  (possible (pos 5) (attr cigarette) (val PallMall))
  (possible (pos 5) (attr cigarette) (val Dunhill))
  (possible (pos 5) (attr cigarette) (val Blend))
  (possible (pos 5) (attr cigarette) (val BlueMaster))
  (possible (pos 5) (attr cigarette) (val Prince))

  ;; --- pet ---
  (possible (pos 1) (attr pet) (val dog))
  (possible (pos 1) (attr pet) (val bird))
  (possible (pos 1) (attr pet) (val cat))
  (possible (pos 1) (attr pet) (val horse))
  (possible (pos 1) (attr pet) (val fish))
  (possible (pos 2) (attr pet) (val dog))
  (possible (pos 2) (attr pet) (val bird))
  (possible (pos 2) (attr pet) (val cat))
  (possible (pos 2) (attr pet) (val horse))
  (possible (pos 2) (attr pet) (val fish))
  (possible (pos 3) (attr pet) (val dog))
  (possible (pos 3) (attr pet) (val bird))
  (possible (pos 3) (attr pet) (val cat))
  (possible (pos 3) (attr pet) (val horse))
  (possible (pos 3) (attr pet) (val fish))
  (possible (pos 4) (attr pet) (val dog))
  (possible (pos 4) (attr pet) (val bird))
  (possible (pos 4) (attr pet) (val cat))
  (possible (pos 4) (attr pet) (val horse))
  (possible (pos 4) (attr pet) (val fish))
  (possible (pos 5) (attr pet) (val dog))
  (possible (pos 5) (attr pet) (val bird))
  (possible (pos 5) (attr pet) (val cat))
  (possible (pos 5) (attr pet) (val horse))
  (possible (pos 5) (attr pet) (val fish))

  ;; --- house convenience facts ---
  (house (position 1))
  (house (position 2))
  (house (position 3))
  (house (position 4))
  (house (position 5)))

;;; =========================================================
;;; RESOLUTION RULES (salience 50)
;;;
;;; Detect when a candidate is uniquely determined and assert
;;; a (resolved) fact. Two triggers:
;;;
;;; A) "Naked single" -- only one value left for a position+attribute
;;; B) "Hidden single" -- only one position left for an attribute+value
;;; =========================================================

(defrule naked-single
  "When only one candidate value remains for a (pos, attr) pair, resolve it"
  (declare (salience 50))
  ;; Exactly one possible fact for this pos+attr
  (possible (pos ?p) (attr ?a) (val ?v))
  (not (possible (pos ?p) (attr ?a) (val ?v2&~?v)))
  ;; Not already resolved
  (not (resolved (pos ?p) (attr ?a)))
  =>
  (assert (resolved (pos ?p) (attr ?a) (val ?v)))
  (printout t "  Resolved: house " ?p " " ?a " = " ?v " (naked single)" crlf))

(defrule hidden-single
  "When a value can only go in one position for a given attribute, resolve it"
  (declare (salience 50))
  ;; Exactly one position where this value is possible for this attribute
  (possible (pos ?p) (attr ?a) (val ?v))
  (not (possible (pos ?p2&~?p) (attr ?a) (val ?v)))
  ;; Not already resolved
  (not (resolved (pos ?p) (attr ?a)))
  =>
  (assert (resolved (pos ?p) (attr ?a) (val ?v)))
  (printout t "  Resolved: house " ?p " " ?a " = " ?v " (hidden single)" crlf))

;;; =========================================================
;;; UNIQUENESS PROPAGATION (salience 40)
;;;
;;; When a value is resolved for a position, eliminate it from
;;; all other positions. Also eliminate all other values for
;;; that position+attribute.
;;; =========================================================

(defrule eliminate-resolved-value-from-other-positions
  "A resolved value cannot appear in any other position for the same attribute"
  (declare (salience 40))
  (resolved (pos ?p) (attr ?a) (val ?v))
  ?cand <- (possible (pos ?p2&~?p) (attr ?a) (val ?v))
  =>
  (retract ?cand))

(defrule eliminate-other-values-from-resolved-position
  "Once resolved, no other value is possible for that position+attribute"
  (declare (salience 40))
  (resolved (pos ?p) (attr ?a) (val ?v))
  ?cand <- (possible (pos ?p) (attr ?a) (val ?v2&~?v))
  =>
  (retract ?cand))

;;; =========================================================
;;; DIRECT CLUE RULES -- ABSOLUTE POSITION (salience 30)
;;;
;;; These clues pin a value to a specific position by
;;; eliminating it from all other positions.
;;; =========================================================

;;; Clue 9: The Norwegian lives in the first house.
(defrule clue-09-norwegian-pos1
  "Norwegian can only be in position 1 -- eliminate from 2-5"
  (declare (salience 30))
  ?cand <- (possible (pos ?p&~1) (attr nationality) (val Norwegian))
  =>
  (retract ?cand))

;;; Clue 8: The person in the center house drinks milk.
(defrule clue-08-milk-pos3
  "Milk can only be in position 3 -- eliminate from 1,2,4,5"
  (declare (salience 30))
  ?cand <- (possible (pos ?p&~3) (attr drink) (val milk))
  =>
  (retract ?cand))

;;; =========================================================
;;; SAME-HOUSE CLUE RULES (salience 20)
;;;
;;; "A and B are in the same house" means:
;;;   - If A is impossible at position P, B is impossible at P.
;;;   - If B is impossible at position P, A is impossible at P.
;;;
;;; CLIPS pattern: bind ?p from the POSITIVE candidate being
;;; eliminated, then test absence of its partner with (not ...).
;;; =========================================================

;;; Clue 1: The Brit lives in the red house.
(defrule clue-01-no-brit-means-no-red
  "If Brit is impossible at P, red is impossible at P"
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr color) (val red))
  (not (possible (pos ?p) (attr nationality) (val Brit)))
  =>
  (retract ?cand))

(defrule clue-01-no-red-means-no-brit
  "If red is impossible at P, Brit is impossible at P"
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr nationality) (val Brit))
  (not (possible (pos ?p) (attr color) (val red)))
  =>
  (retract ?cand))

;;; Clue 2: The Swede keeps dogs.
(defrule clue-02-no-swede-means-no-dog
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr pet) (val dog))
  (not (possible (pos ?p) (attr nationality) (val Swede)))
  =>
  (retract ?cand))

(defrule clue-02-no-dog-means-no-swede
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr nationality) (val Swede))
  (not (possible (pos ?p) (attr pet) (val dog)))
  =>
  (retract ?cand))

;;; Clue 3: The Dane drinks tea.
(defrule clue-03-no-dane-means-no-tea
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr drink) (val tea))
  (not (possible (pos ?p) (attr nationality) (val Dane)))
  =>
  (retract ?cand))

(defrule clue-03-no-tea-means-no-dane
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr nationality) (val Dane))
  (not (possible (pos ?p) (attr drink) (val tea)))
  =>
  (retract ?cand))

;;; Clue 5: The green house owner drinks coffee.
(defrule clue-05-no-green-means-no-coffee
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr drink) (val coffee))
  (not (possible (pos ?p) (attr color) (val green)))
  =>
  (retract ?cand))

(defrule clue-05-no-coffee-means-no-green
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr color) (val green))
  (not (possible (pos ?p) (attr drink) (val coffee)))
  =>
  (retract ?cand))

;;; Clue 6: The Pall Mall smoker keeps birds.
(defrule clue-06-no-pallmall-means-no-bird
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr pet) (val bird))
  (not (possible (pos ?p) (attr cigarette) (val PallMall)))
  =>
  (retract ?cand))

(defrule clue-06-no-bird-means-no-pallmall
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr cigarette) (val PallMall))
  (not (possible (pos ?p) (attr pet) (val bird)))
  =>
  (retract ?cand))

;;; Clue 7: The owner of the yellow house smokes Dunhill.
(defrule clue-07-no-yellow-means-no-dunhill
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr cigarette) (val Dunhill))
  (not (possible (pos ?p) (attr color) (val yellow)))
  =>
  (retract ?cand))

(defrule clue-07-no-dunhill-means-no-yellow
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr color) (val yellow))
  (not (possible (pos ?p) (attr cigarette) (val Dunhill)))
  =>
  (retract ?cand))

;;; Clue 12: The Blue Master smoker drinks beer.
(defrule clue-12-no-bluemaster-means-no-beer
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr drink) (val beer))
  (not (possible (pos ?p) (attr cigarette) (val BlueMaster)))
  =>
  (retract ?cand))

(defrule clue-12-no-beer-means-no-bluemaster
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr cigarette) (val BlueMaster))
  (not (possible (pos ?p) (attr drink) (val beer)))
  =>
  (retract ?cand))

;;; Clue 13: The German smokes Prince.
(defrule clue-13-no-german-means-no-prince
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr cigarette) (val Prince))
  (not (possible (pos ?p) (attr nationality) (val German)))
  =>
  (retract ?cand))

(defrule clue-13-no-prince-means-no-german
  (declare (salience 20))
  ?cand <- (possible (pos ?p) (attr nationality) (val German))
  (not (possible (pos ?p) (attr cigarette) (val Prince)))
  =>
  (retract ?cand))

;;; =========================================================
;;; LEFT-OF CLUE RULES (salience 20)
;;;
;;; Clue 4: The green house is immediately to the left of the
;;; white house.  green(P) => white(P+1), white(P) => green(P-1).
;;; =========================================================

;;; Green cannot be position 5 (nothing to its right for white).
(defrule clue-04-green-not-pos5
  "Green cannot be in position 5"
  (declare (salience 30))
  ?cand <- (possible (pos 5) (attr color) (val green))
  =>
  (retract ?cand))

;;; White cannot be position 1 (nothing to its left for green).
(defrule clue-04-white-not-pos1
  "White cannot be in position 1"
  (declare (salience 30))
  ?cand <- (possible (pos 1) (attr color) (val white))
  =>
  (retract ?cand))

;;; If green is impossible at P, then white is impossible at P+1.
(defrule clue-04-no-green-means-no-white-right
  "If green can't be at P, then white can't be at P+1"
  (declare (salience 20))
  ?cand <- (possible (pos ?p2) (attr color) (val white))
  (test (>= ?p2 2))
  (not (possible (pos ?p1&:(= ?p1 (- ?p2 1))) (attr color) (val green)))
  =>
  (retract ?cand))

;;; If white is impossible at P+1, then green is impossible at P.
(defrule clue-04-no-white-means-no-green-left
  "If white can't be at P+1, then green can't be at P"
  (declare (salience 20))
  ?cand <- (possible (pos ?p1) (attr color) (val green))
  (test (<= ?p1 4))
  (not (possible (pos ?p2&:(= ?p2 (+ ?p1 1))) (attr color) (val white)))
  =>
  (retract ?cand))

;;; =========================================================
;;; NEXT-TO CLUE RULES (salience 15)
;;;
;;; "A is next to B" means they are in adjacent positions.
;;; If A can't have any neighbor with B, eliminate A from that
;;; position -- and vice versa.
;;;
;;; Pattern: bind ?p from the candidate being tested, then
;;; check that NEITHER neighbor has the partner value possible.
;;; =========================================================

;;; Clue 14: The Norwegian lives next to the blue house.
;;; Norwegian is resolved to position 1 by clue 9.
;;; Therefore blue must be position 2.
(defrule clue-14-blue-adj-to-norwegian
  "Blue must be adjacent to Norwegian. Since Norwegian is at 1, blue is at 2."
  (declare (salience 20))
  (resolved (pos 1) (attr nationality) (val Norwegian))
  ?cand <- (possible (pos ?p&~2) (attr color) (val blue))
  =>
  (retract ?cand))

;;; Clue 10: The Blend smoker lives next to the cat owner.
(defrule clue-10-blend-needs-adj-cat
  "If cat is impossible at both neighbors of P, Blend is impossible at P"
  (declare (salience 15))
  ?cand <- (possible (pos ?p) (attr cigarette) (val Blend))
  (not (possible (pos ?pL&:(= ?pL (- ?p 1))) (attr pet) (val cat)))
  (not (possible (pos ?pR&:(= ?pR (+ ?p 1))) (attr pet) (val cat)))
  =>
  (retract ?cand))

(defrule clue-10-cat-needs-adj-blend
  "If Blend is impossible at both neighbors of P, cat is impossible at P"
  (declare (salience 15))
  ?cand <- (possible (pos ?p) (attr pet) (val cat))
  (not (possible (pos ?pL&:(= ?pL (- ?p 1))) (attr cigarette) (val Blend)))
  (not (possible (pos ?pR&:(= ?pR (+ ?p 1))) (attr cigarette) (val Blend)))
  =>
  (retract ?cand))

;;; Clue 11: The person who keeps horses lives next to the Dunhill smoker.
(defrule clue-11-horse-needs-adj-dunhill
  "If Dunhill is impossible at both neighbors of P, horse is impossible at P"
  (declare (salience 15))
  ?cand <- (possible (pos ?p) (attr pet) (val horse))
  (not (possible (pos ?pL&:(= ?pL (- ?p 1))) (attr cigarette) (val Dunhill)))
  (not (possible (pos ?pR&:(= ?pR (+ ?p 1))) (attr cigarette) (val Dunhill)))
  =>
  (retract ?cand))

(defrule clue-11-dunhill-needs-adj-horse
  "If horse is impossible at both neighbors of P, Dunhill is impossible at P"
  (declare (salience 15))
  ?cand <- (possible (pos ?p) (attr cigarette) (val Dunhill))
  (not (possible (pos ?pL&:(= ?pL (- ?p 1))) (attr pet) (val horse)))
  (not (possible (pos ?pR&:(= ?pR (+ ?p 1))) (attr pet) (val horse)))
  =>
  (retract ?cand))

;;; Clue 15: The Blend smoker has a neighbor who drinks water.
(defrule clue-15-blend-needs-adj-water
  "If water is impossible at both neighbors of P, Blend is impossible at P"
  (declare (salience 15))
  ?cand <- (possible (pos ?p) (attr cigarette) (val Blend))
  (not (possible (pos ?pL&:(= ?pL (- ?p 1))) (attr drink) (val water)))
  (not (possible (pos ?pR&:(= ?pR (+ ?p 1))) (attr drink) (val water)))
  =>
  (retract ?cand))

(defrule clue-15-water-needs-adj-blend
  "If Blend is impossible at both neighbors of P, water is impossible at P"
  (declare (salience 15))
  ?cand <- (possible (pos ?p) (attr drink) (val water))
  (not (possible (pos ?pL&:(= ?pL (- ?p 1))) (attr cigarette) (val Blend)))
  (not (possible (pos ?pR&:(= ?pR (+ ?p 1))) (attr cigarette) (val Blend)))
  =>
  (retract ?cand))

;;; =========================================================
;;; SAME-HOUSE FORCING (salience 10)
;;;
;;; Stronger inference: when a value is RESOLVED at position P,
;;; its same-house partner must also be at P. Eliminate the
;;; partner from all other positions.
;;; =========================================================

;;; Clue 1: Brit <-> red
(defrule clue-01-brit-resolved-forces-red
  (declare (salience 10))
  (resolved (pos ?p) (attr nationality) (val Brit))
  ?cand <- (possible (pos ?p2&~?p) (attr color) (val red))
  =>
  (retract ?cand))

(defrule clue-01-red-resolved-forces-brit
  (declare (salience 10))
  (resolved (pos ?p) (attr color) (val red))
  ?cand <- (possible (pos ?p2&~?p) (attr nationality) (val Brit))
  =>
  (retract ?cand))

;;; Clue 2: Swede <-> dog
(defrule clue-02-swede-resolved-forces-dog
  (declare (salience 10))
  (resolved (pos ?p) (attr nationality) (val Swede))
  ?cand <- (possible (pos ?p2&~?p) (attr pet) (val dog))
  =>
  (retract ?cand))

(defrule clue-02-dog-resolved-forces-swede
  (declare (salience 10))
  (resolved (pos ?p) (attr pet) (val dog))
  ?cand <- (possible (pos ?p2&~?p) (attr nationality) (val Swede))
  =>
  (retract ?cand))

;;; Clue 3: Dane <-> tea
(defrule clue-03-dane-resolved-forces-tea
  (declare (salience 10))
  (resolved (pos ?p) (attr nationality) (val Dane))
  ?cand <- (possible (pos ?p2&~?p) (attr drink) (val tea))
  =>
  (retract ?cand))

(defrule clue-03-tea-resolved-forces-dane
  (declare (salience 10))
  (resolved (pos ?p) (attr drink) (val tea))
  ?cand <- (possible (pos ?p2&~?p) (attr nationality) (val Dane))
  =>
  (retract ?cand))

;;; Clue 5: green <-> coffee
(defrule clue-05-green-resolved-forces-coffee
  (declare (salience 10))
  (resolved (pos ?p) (attr color) (val green))
  ?cand <- (possible (pos ?p2&~?p) (attr drink) (val coffee))
  =>
  (retract ?cand))

(defrule clue-05-coffee-resolved-forces-green
  (declare (salience 10))
  (resolved (pos ?p) (attr drink) (val coffee))
  ?cand <- (possible (pos ?p2&~?p) (attr color) (val green))
  =>
  (retract ?cand))

;;; Clue 6: PallMall <-> bird
(defrule clue-06-pallmall-resolved-forces-bird
  (declare (salience 10))
  (resolved (pos ?p) (attr cigarette) (val PallMall))
  ?cand <- (possible (pos ?p2&~?p) (attr pet) (val bird))
  =>
  (retract ?cand))

(defrule clue-06-bird-resolved-forces-pallmall
  (declare (salience 10))
  (resolved (pos ?p) (attr pet) (val bird))
  ?cand <- (possible (pos ?p2&~?p) (attr cigarette) (val PallMall))
  =>
  (retract ?cand))

;;; Clue 7: yellow <-> Dunhill
(defrule clue-07-yellow-resolved-forces-dunhill
  (declare (salience 10))
  (resolved (pos ?p) (attr color) (val yellow))
  ?cand <- (possible (pos ?p2&~?p) (attr cigarette) (val Dunhill))
  =>
  (retract ?cand))

(defrule clue-07-dunhill-resolved-forces-yellow
  (declare (salience 10))
  (resolved (pos ?p) (attr cigarette) (val Dunhill))
  ?cand <- (possible (pos ?p2&~?p) (attr color) (val yellow))
  =>
  (retract ?cand))

;;; Clue 12: BlueMaster <-> beer
(defrule clue-12-bluemaster-resolved-forces-beer
  (declare (salience 10))
  (resolved (pos ?p) (attr cigarette) (val BlueMaster))
  ?cand <- (possible (pos ?p2&~?p) (attr drink) (val beer))
  =>
  (retract ?cand))

(defrule clue-12-beer-resolved-forces-bluemaster
  (declare (salience 10))
  (resolved (pos ?p) (attr drink) (val beer))
  ?cand <- (possible (pos ?p2&~?p) (attr cigarette) (val BlueMaster))
  =>
  (retract ?cand))

;;; Clue 13: German <-> Prince
(defrule clue-13-german-resolved-forces-prince
  (declare (salience 10))
  (resolved (pos ?p) (attr nationality) (val German))
  ?cand <- (possible (pos ?p2&~?p) (attr cigarette) (val Prince))
  =>
  (retract ?cand))

(defrule clue-13-prince-resolved-forces-german
  (declare (salience 10))
  (resolved (pos ?p) (attr cigarette) (val Prince))
  ?cand <- (possible (pos ?p2&~?p) (attr nationality) (val German))
  =>
  (retract ?cand))

;;; =========================================================
;;; HOUSE VIEW UPDATES (salience -10)
;;;
;;; Populate the convenience (house ...) facts as the solver
;;; resolves each attribute.
;;; =========================================================

(defrule update-house-nationality
  (declare (salience -10))
  (resolved (pos ?p) (attr nationality) (val ?v))
  ?h <- (house (position ?p) (nationality unknown))
  =>
  (modify ?h (nationality ?v)))

(defrule update-house-color
  (declare (salience -10))
  (resolved (pos ?p) (attr color) (val ?v))
  ?h <- (house (position ?p) (color unknown))
  =>
  (modify ?h (color ?v)))

(defrule update-house-drink
  (declare (salience -10))
  (resolved (pos ?p) (attr drink) (val ?v))
  ?h <- (house (position ?p) (drink unknown))
  =>
  (modify ?h (drink ?v)))

(defrule update-house-cigarette
  (declare (salience -10))
  (resolved (pos ?p) (attr cigarette) (val ?v))
  ?h <- (house (position ?p) (cigarette unknown))
  =>
  (modify ?h (cigarette ?v)))

(defrule update-house-pet
  (declare (salience -10))
  (resolved (pos ?p) (attr pet) (val ?v))
  ?h <- (house (position ?p) (pet unknown))
  =>
  (modify ?h (pet ?v)))

;;; =========================================================
;;; SOLUTION EXTRACTION (salience -50)
;;; =========================================================

(defrule initialize-solution
  "Create the solution tracker at startup"
  (declare (salience 100))
  (not (solution))
  =>
  (assert (solution (fish-owner unknown))))

(defrule find-fish-owner
  "When fish is resolved, record who owns it"
  (declare (salience -50))
  (resolved (pos ?p) (attr pet) (val fish))
  (resolved (pos ?p) (attr nationality) (val ?n))
  ?sol <- (solution (fish-owner unknown))
  =>
  (modify ?sol (fish-owner ?n))
  (printout t crlf)
  (printout t "========================================" crlf)
  (printout t "  SOLUTION: The " ?n " owns the fish!" crlf)
  (printout t "========================================" crlf))

;;; =========================================================
;;; DISPLAY RULES (salience -100)
;;; =========================================================

(defrule print-all-houses
  "Print the complete solution grid"
  (declare (salience -100))
  (solution (fish-owner ?owner&~unknown))
  (house (position 1) (nationality ?n1&~unknown) (color ?c1&~unknown)
         (drink ?d1&~unknown) (cigarette ?cig1&~unknown) (pet ?p1&~unknown))
  (house (position 2) (nationality ?n2&~unknown) (color ?c2&~unknown)
         (drink ?d2&~unknown) (cigarette ?cig2&~unknown) (pet ?p2&~unknown))
  (house (position 3) (nationality ?n3&~unknown) (color ?c3&~unknown)
         (drink ?d3&~unknown) (cigarette ?cig3&~unknown) (pet ?p3&~unknown))
  (house (position 4) (nationality ?n4&~unknown) (color ?c4&~unknown)
         (drink ?d4&~unknown) (cigarette ?cig4&~unknown) (pet ?p4&~unknown))
  (house (position 5) (nationality ?n5&~unknown) (color ?c5&~unknown)
         (drink ?d5&~unknown) (cigarette ?cig5&~unknown) (pet ?p5&~unknown))
  =>
  (printout t crlf "--- Final House Assignments ---" crlf)
  (printout t "Pos | Nationality | Color  | Drink  | Cigarette  | Pet" crlf)
  (printout t "----+-------------+--------+--------+------------+------" crlf)
  (printout t " 1  | " ?n1 " | " ?c1 " | " ?d1 " | " ?cig1 " | " ?p1 crlf)
  (printout t " 2  | " ?n2 " | " ?c2 " | " ?d2 " | " ?cig2 " | " ?p2 crlf)
  (printout t " 3  | " ?n3 " | " ?c3 " | " ?d3 " | " ?cig3 " | " ?p3 crlf)
  (printout t " 4  | " ?n4 " | " ?c4 " | " ?d4 " | " ?cig4 " | " ?p4 crlf)
  (printout t " 5  | " ?n5 " | " ?c5 " | " ?d5 " | " ?cig5 " | " ?p5 crlf)
  (printout t crlf))

;;; =========================================================
;;; END OF RULES
;;; =========================================================
