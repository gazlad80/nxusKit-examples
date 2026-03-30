;;; scoring-adjustments.clp - Scoring bonus/penalty rules for musicality dimensions

;;; === RESOLUTION QUALITY ===

;;; Penalty: Unresolved tritone
(defrule tritone-unresolved-penalty
  "Penalize tritones that don't resolve"
  (dissonance (from-note ?i) (to-note ?j) (resolved no))
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(= (mod (abs ?s) 12) 6)))
  =>
  (assert (scoring-adjustment (dimension resolution) (amount -15)
    (reason (str-cat "Tritone at notes " ?i "-" ?j " lacks resolution")))))

;;; Bonus: Resolved dissonance
(defrule resolved-dissonance-bonus
  "Bonus for properly resolved dissonances"
  (dissonance (from-note ?i) (to-note ?j) (resolved yes))
  =>
  (assert (scoring-adjustment (dimension resolution) (amount 5)
    (reason (str-cat "Dissonance at notes " ?i "-" ?j " resolves well")))))

;;; === MELODIC INTEREST ===

;;; Penalty: Large leap without recovery
(defrule large-leap-no-recovery
  "Penalize large leaps (>7 semitones) not followed by contrary motion"
  (interval (from-index ?i) (to-index ?j) (semitones ?s1&:(> (abs ?s1) 7)))
  (interval (from-index ?j) (to-index ?k) (semitones ?s2))
  (test (or (and (> ?s1 0) (>= ?s2 0)) (and (< ?s1 0) (<= ?s2 0))))
  =>
  (assert (scoring-adjustment (dimension melody) (amount -8)
    (reason (str-cat "Large leap at notes " ?i "-" ?j " not recovered by contrary motion")))))

;;; Bonus: Large leap with step recovery
(defrule large-leap-step-recovery
  "Bonus for large leaps followed by stepwise contrary motion"
  (interval (from-index ?i) (to-index ?j) (semitones ?s1&:(> (abs ?s1) 7)))
  (interval (from-index ?j) (to-index ?k) (semitones ?s2&:(and (<= (abs ?s2) 2) (> (abs ?s2) 0))))
  (test (or (and (> ?s1 0) (< ?s2 0)) (and (< ?s1 0) (> ?s2 0))))
  =>
  (assert (scoring-adjustment (dimension melody) (amount 5)
    (reason (str-cat "Large leap at notes " ?i "-" ?j " recovered by step")))))

;;; Penalty: Repeated notes (static melody)
(defrule repeated-notes-penalty
  "Penalize three or more repeated pitches"
  (interval (from-index ?i) (to-index ?j) (semitones 0))
  (interval (from-index ?j) (to-index ?k) (semitones 0))
  =>
  (assert (scoring-adjustment (dimension melody) (amount -5)
    (reason (str-cat "Repeated pitches at notes " ?i "-" ?k " reduce melodic interest")))))

;;; === HARMONIC COHERENCE ===

;;; Penalty: Out-of-scale notes
(defrule out-of-scale-penalty
  "Penalize notes that don't fit the detected scale"
  (scale-membership (note-index ?i) (in-scale no))
  =>
  (assert (scoring-adjustment (dimension harmony) (amount -3)
    (reason (str-cat "Note " ?i " is outside the detected scale")))))

;;; Bonus: Strong scale adherence
(defrule high-scale-adherence-bonus
  "Bonus when all notes are in scale"
  (declare (salience -20))
  (not (scale-membership (in-scale no)))
  =>
  (assert (scoring-adjustment (dimension harmony) (amount 10)
    (reason "All notes fit within the detected scale"))))

;;; === RHYTHMIC VARIETY ===

;;; Penalty: Very low rhythmic entropy
(defrule low-rhythmic-variety-penalty
  "Penalize repetitive rhythms"
  (rhythmic-entropy (value ?v&:(< ?v 0.3)) (unique-durations ?u&:(<= ?u 1)))
  =>
  (assert (scoring-adjustment (dimension rhythm) (amount -10)
    (reason "Rhythmic variety is very low - durations are repetitive"))))

;;; Bonus: Good rhythmic variety
(defrule good-rhythmic-variety-bonus
  "Bonus for rhythmic diversity"
  (rhythmic-entropy (value ?v&:(> ?v 0.7)) (unique-durations ?u&:(>= ?u 3)))
  =>
  (assert (scoring-adjustment (dimension rhythm) (amount 8)
    (reason "Good rhythmic variety with diverse note durations"))))

;;; === DYNAMICS EXPRESSION ===

;;; Penalty: Flat dynamics
(defrule flat-dynamics-penalty
  "Penalize sequences with no velocity variation"
  (dynamics-summary (velocity-range ?r&:(< ?r 10)))
  =>
  (assert (scoring-adjustment (dimension dynamics) (amount -8)
    (reason "Velocity is flat - no dynamic expression"))))

;;; Bonus: Expressive dynamics
(defrule expressive-dynamics-bonus
  "Bonus for good velocity range"
  (dynamics-summary (velocity-range ?r&:(> ?r 40)) (velocity-std ?s&:(> ?s 15.0)))
  =>
  (assert (scoring-adjustment (dimension dynamics) (amount 8)
    (reason "Good dynamic range and expression"))))

;;; === STRUCTURAL BALANCE ===

;;; Bonus: Consistent phrase lengths (assessed via contour changes)
(defrule balanced-contour-bonus
  "Bonus for balanced melodic contour"
  (contour (type ?t&:(or (eq ?t arch) (eq ?t inverse-arch))) (direction-changes ?d&:(and (> ?d 1) (< ?d 6))))
  =>
  (assert (scoring-adjustment (dimension structure) (amount 5)
    (reason "Melodic contour shows good arch structure"))))
