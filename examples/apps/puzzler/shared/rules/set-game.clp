;;;======================================================
;;; Set Game Rules
;;;
;;; CLIPS rules for finding valid sets in the Set card game.
;;; A valid set consists of three cards where each attribute
;;; (shape, color, count, shading) is either:
;;; - All the same across the three cards, OR
;;; - All different across the three cards
;;;
;;; Rules:
;;; 1. Pattern matching to identify potential sets
;;; 2. Validation of all four attributes
;;; 3. Recording of valid sets found
;;;======================================================

;;; Template for a Set Game card
(deftemplate set-card
   "A single Set Game card"
   (slot id (type INTEGER))
   (slot shape (type SYMBOL) (allowed-symbols diamond oval squiggle))
   (slot color (type SYMBOL) (allowed-symbols red green purple))
   (slot count (type INTEGER) (range 1 3))
   (slot shading (type SYMBOL) (allowed-symbols solid striped empty)))

;;; Template for tracking found valid sets
(deftemplate valid-set
   "A valid set of three cards"
   (slot card1-id (type INTEGER))
   (slot card2-id (type INTEGER))
   (slot card3-id (type INTEGER))
   (slot shape-match (type SYMBOL) (allowed-symbols all-same all-different))
   (slot color-match (type SYMBOL) (allowed-symbols all-same all-different))
   (slot count-match (type SYMBOL) (allowed-symbols all-same all-different))
   (slot shading-match (type SYMBOL) (allowed-symbols all-same all-different)))

;;; Template for game state
(deftemplate game-state
   "Current state of the game"
   (slot total-cards (type INTEGER))
   (slot sets-found (type INTEGER) (default 0))
   (slot iterations (type INTEGER) (default 0)))

;;; Template for requesting LLM help
(deftemplate llm-help-request
   "Request for LLM assistance when stuck"
   (slot context (type STRING))
   (slot cards-description (type STRING)))

;;;======================================================
;;; Helper Functions
;;;======================================================

(deffunction attribute-valid (?a ?b ?c)
   "Check if three values satisfy the set rule (all same or all different)"
   (or (and (eq ?a ?b) (eq ?b ?c))          ; All same
       (and (neq ?a ?b) (neq ?b ?c) (neq ?a ?c))))  ; All different

(deffunction get-match-type (?a ?b ?c)
   "Return the match type for an attribute"
   (if (and (eq ?a ?b) (eq ?b ?c))
      then all-same
      else all-different))

;;;======================================================
;;; Set Detection Rules
;;;======================================================

;;; Main rule to find valid sets
;;; This rule matches all combinations of 3 cards and checks validity
(defrule find-valid-set
   "Find a valid set among three cards"
   (declare (salience 50))

   ;; Match three different cards (ordered by ID to avoid duplicates)
   (set-card (id ?id1) (shape ?s1) (color ?c1) (count ?n1) (shading ?sh1))
   (set-card (id ?id2&:(> ?id2 ?id1)) (shape ?s2) (color ?c2) (count ?n2) (shading ?sh2))
   (set-card (id ?id3&:(> ?id3 ?id2)) (shape ?s3) (color ?c3) (count ?n3) (shading ?sh3))

   ;; Ensure this set hasn't been found already
   (not (valid-set (card1-id ?id1) (card2-id ?id2) (card3-id ?id3)))

   ;; Check all attributes satisfy set rules
   (test (attribute-valid ?s1 ?s2 ?s3))
   (test (attribute-valid ?c1 ?c2 ?c3))
   (test (attribute-valid ?n1 ?n2 ?n3))
   (test (attribute-valid ?sh1 ?sh2 ?sh3))

   ;; Get the game state to update
   ?state <- (game-state (sets-found ?found) (iterations ?iter))
   =>
   ;; Record the valid set
   (assert (valid-set
      (card1-id ?id1)
      (card2-id ?id2)
      (card3-id ?id3)
      (shape-match (get-match-type ?s1 ?s2 ?s3))
      (color-match (get-match-type ?c1 ?c2 ?c3))
      (count-match (get-match-type ?n1 ?n2 ?n3))
      (shading-match (get-match-type ?sh1 ?sh2 ?sh3))))

   ;; Update state
   (modify ?state (sets-found (+ ?found 1)) (iterations (+ ?iter 1))))

;;;======================================================
;;; Completion Rules
;;;======================================================

(defrule no-sets-found
   "Detect when no valid sets exist in the hand"
   (declare (salience -100))
   (game-state (total-cards ?total&:(>= ?total 3)) (sets-found 0))
   (not (llm-help-request))
   =>
   (assert (llm-help-request
      (context "No valid sets found by CLIPS rules")
      (cards-description "Review cards for missed sets"))))

;;;======================================================
;;; Query Functions
;;;======================================================

(deffunction count-valid-sets ()
   "Count the number of valid sets found"
   (bind ?count 0)
   (do-for-all-facts ((?v valid-set)) TRUE
      (bind ?count (+ ?count 1)))
   ?count)

(deffunction get-all-sets ()
   "Return all valid sets as a formatted string"
   (bind ?result "")
   (do-for-all-facts ((?v valid-set)) TRUE
      (bind ?result (str-cat ?result
         "Set: [" ?v:card1-id ", " ?v:card2-id ", " ?v:card3-id "] "
         "(shape:" ?v:shape-match " color:" ?v:color-match
         " count:" ?v:count-match " shading:" ?v:shading-match ")\n")))
   ?result)
