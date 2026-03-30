;;; suggestions.clp - Context-aware improvement recommendation rules

;;; === MELODIC SUGGESTIONS ===

;;; Suggest passing tone for large intervals
(defrule suggest-passing-tone
  "Suggest adding passing tones for intervals > perfect 4th"
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(> (abs ?s) 5)))
  =>
  (assert (suggestion (category melody) (severity suggestion) (note-index ?i)
    (message (str-cat "Consider adding a passing tone between notes " ?i " and " ?j " to smooth the leap")))))

;;; Suggest neighbor tones for static passages
(defrule suggest-neighbor-tone
  "Suggest neighbor tones when melody is too static"
  (interval (from-index ?i) (to-index ?j) (semitones 0))
  =>
  (assert (suggestion (category melody) (severity info) (note-index ?i)
    (message (str-cat "Consider adding a neighbor tone between repeated notes " ?i " and " ?j)))))

;;; Suggest direction change after long runs
(defrule suggest-direction-change
  "Suggest changing direction after extended runs"
  (interval (from-index ?i) (to-index ?j) (semitones ?s1&:(> ?s1 0)))
  (interval (from-index ?j) (to-index ?k) (semitones ?s2&:(> ?s2 0)))
  (interval (from-index ?k) (to-index ?l) (semitones ?s3&:(> ?s3 0)))
  =>
  (assert (suggestion (category melody) (severity suggestion) (note-index ?l)
    (message (str-cat "Consider a direction change after note " ?l " - extended ascending run detected")))))

;;; === RESOLUTION SUGGESTIONS ===

;;; Suggest resolution for dissonance
(defrule suggest-resolve-dissonance
  "Suggest resolving hanging dissonances"
  (dissonance (to-note ?j) (resolved no))
  =>
  (assert (suggestion (category resolution) (severity suggestion) (note-index ?j)
    (message (str-cat "Dissonance at note " ?j " would benefit from resolution to a consonant interval")))))

;;; Suggest leading tone treatment
(defrule suggest-leading-tone
  "Suggest proper leading tone treatment"
  (context (detected-mode major))
  (note (index ?i) (pitch-class ?pc))
  (not (note (index ?j&:(= ?j (+ ?i 1)))))
  =>
  (assert (suggestion (category resolution) (severity info) (note-index ?i)
    (message "If this is a leading tone, ensure it resolves upward to the tonic"))))

;;; === RHYTHM SUGGESTIONS ===

;;; Suggest rhythmic variation
(defrule suggest-vary-rhythm
  "Suggest adding rhythmic variation when durations are uniform"
  (rhythmic-entropy (value ?e&:(< ?e 0.5)) (unique-durations ?u&:(<= ?u 2)))
  =>
  (assert (suggestion (category rhythm) (severity suggestion) (note-index 0)
    (message "Consider adding rhythmic variation - try mixing quarter notes with eighths or dotted rhythms"))))

;;; Suggest syncopation
(defrule suggest-syncopation
  "Suggest syncopation for more interest"
  (rhythmic-entropy (value ?e&:(< ?e 0.4)))
  =>
  (assert (suggestion (category rhythm) (severity info) (note-index 0)
    (message "Consider adding syncopation by accenting off-beats"))))

;;; === DYNAMICS SUGGESTIONS ===

;;; Suggest dynamic variation
(defrule suggest-dynamics
  "Suggest adding dynamic variation"
  (dynamics-summary (velocity-range ?r&:(< ?r 20)))
  =>
  (assert (suggestion (category dynamics) (severity suggestion) (note-index 0)
    (message "Add dynamic variation - try crescendo/decrescendo or accent important notes"))))

;;; Suggest phrase dynamics
(defrule suggest-phrase-dynamics
  "Suggest shaping dynamics to phrases"
  (dynamics-summary (velocity-std ?s&:(< ?s 10.0)) (velocity-range ?r&:(< ?r 30)))
  =>
  (assert (suggestion (category dynamics) (severity info) (note-index 0)
    (message "Consider shaping dynamics to follow phrase structure - louder at climax, softer at resolution"))))

;;; === HARMONY SUGGESTIONS ===

;;; Suggest chromatic passing tones carefully
(defrule suggest-chromatic-use
  "Advise on chromatic note usage"
  (scale-membership (note-index ?i) (in-scale no))
  (interval (from-index ?i) (to-index ?j) (semitones ?s&:(and (>= (abs ?s) 1) (<= (abs ?s) 2))))
  =>
  (assert (suggestion (category harmony) (severity info) (note-index ?i)
    (message (str-cat "Chromatic note at " ?i " works as a passing/neighbor tone - good usage")))))

;;; Suggest scale adherence for beginners
(defrule suggest-scale-adherence
  "Suggest staying in scale for cleaner sound"
  (scale-membership (note-index ?i) (in-scale no))
  (not (interval (from-index ?i) (semitones ?s&:(and (>= (abs ?s) 1) (<= (abs ?s) 2)))))
  =>
  (assert (suggestion (category harmony) (severity warning) (note-index ?i)
    (message (str-cat "Note " ?i " is outside the scale and doesn't appear to be a passing tone - consider adjusting")))))

;;; === STRUCTURE SUGGESTIONS ===

;;; Suggest phrase balance
(defrule suggest-phrase-balance
  "Suggest improving phrase structure"
  (contour (type static))
  =>
  (assert (suggestion (category structure) (severity suggestion) (note-index 0)
    (message "Consider adding melodic shape - try an arch contour with a clear climax and resolution"))))

;;; Suggest motivic development
(defrule suggest-motivic-development
  "Suggest developing melodic motifs"
  (contour (direction-changes ?d&:(> ?d 8)))
  =>
  (assert (suggestion (category structure) (severity info) (note-index 0)
    (message "Many direction changes detected - consider establishing a clearer melodic motif and developing it"))))
