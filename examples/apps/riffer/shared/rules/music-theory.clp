;;; music-theory.clp - Core music theory rules for interval and dissonance detection

;;; Identify dissonant intervals and track them for resolution
(defrule identify-strong-dissonance
  "Mark strong dissonances (minor 2nd, tritone, major 7th)"
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(or (= (mod (abs ?s) 12) 1)
                                                              (= (mod (abs ?s) 12) 6)
                                                              (= (mod (abs ?s) 12) 11)))
            (quality strong-dissonance))
  =>
  (assert (dissonance (interval-index ?i) (from-note ?i) (to-note ?j) (resolved pending))))

;;; Check if dissonance resolves to consonance
(defrule check-dissonance-resolution
  "Check if a dissonance is followed by consonance within 2 notes"
  ?d <- (dissonance (to-note ?j) (resolved pending))
  (interval (from-index ?j) (quality ?q&:(or (eq ?q perfect-consonance) (eq ?q imperfect-consonance))))
  =>
  (modify ?d (resolved yes)))

;;; Mark unresolved dissonances after all resolution checks
(defrule mark-unresolved-dissonance
  "Mark dissonances as unresolved if not resolved within window"
  (declare (salience -10))
  ?d <- (dissonance (to-note ?j) (resolved pending))
  (not (interval (from-index ?j)))
  =>
  (modify ?d (resolved no)))

;;; Detect stepwise motion (conjunct melody) - reward smooth voice leading
(defrule detect-stepwise-motion
  "Reward smooth stepwise motion"
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(and (>= (abs ?s) 1) (<= (abs ?s) 2))))
  =>
  (assert (scoring-adjustment (dimension melody) (amount 2)
    (reason "Smooth stepwise motion"))))

;;; Detect large leaps (6 semitones or more)
(defrule detect-large-leap
  "Note large leaps - can be effective but need recovery"
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(>= (abs ?s) 7)))
  =>
  (assert (suggestion (category melody) (severity info) (note-index ?i)
    (message "Large melodic leap detected - ensure melodic recovery follows"))))

;;; Detect octave leap specifically
(defrule detect-octave-leap
  "Note octave leaps"
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(and (= (mod (abs ?s) 12) 0) (> (abs ?s) 0))))
  =>
  (assert (suggestion (category melody) (severity info) (note-index ?i)
    (message "Octave leap detected - ensure it serves a musical purpose"))))

;;; Perfect consonance at phrase boundaries
(defrule phrase-ends-on-consonance
  "Reward sequences that end on perfect consonance"
  (declare (salience -5))
  (note (index ?last))
  (not (note (index ?n&:(> ?n ?last))))
  (interval (to-index ?last) (quality perfect-consonance))
  =>
  (assert (scoring-adjustment (dimension resolution) (amount 8)
    (reason "Phrase ends with perfect consonance"))))

;;; Detect ascending motion tendency
(defrule detect-ascending-pattern
  "Track ascending melodic pattern"
  (interval (from-index ?i) (to-index ?j) (semitones ?s1&:(> ?s1 0)))
  (interval (from-index ?j) (to-index ?k) (semitones ?s2&:(> ?s2 0)))
  =>
  (assert (suggestion (category melody) (severity info) (note-index ?j)
    (message "Consistent ascending motion detected"))))

;;; Detect descending motion tendency
(defrule detect-descending-pattern
  "Track descending melodic pattern"
  (interval (from-index ?i) (to-index ?j) (semitones ?s1&:(< ?s1 0)))
  (interval (from-index ?j) (to-index ?k) (semitones ?s2&:(< ?s2 0)))
  =>
  (assert (suggestion (category melody) (severity info) (note-index ?j)
    (message "Consistent descending motion detected"))))
