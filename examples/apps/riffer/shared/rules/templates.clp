;;; templates.clp - Music theory fact templates for Riffer
;;; These templates define the structure of facts used in music analysis

;;; Note representation
(deftemplate note
  "A single musical note in a sequence"
  (slot index (type INTEGER) (default 0))
  (slot pitch (type INTEGER) (range 0 127) (default 60))
  (slot pitch-class (type SYMBOL) (allowed-symbols C Cs D Ds E F Fs G Gs A As B))
  (slot octave (type INTEGER) (range -1 9) (default 4))
  (slot duration (type INTEGER) (default 480))
  (slot velocity (type INTEGER) (range 0 127) (default 80))
  (slot start-tick (type INTEGER) (default 0)))

;;; Interval between two consecutive notes
(deftemplate interval
  "Interval between two notes"
  (slot from-index (type INTEGER))
  (slot to-index (type INTEGER))
  (slot semitones (type INTEGER) (range -127 127))
  (slot name (type STRING))
  (slot quality (type SYMBOL) (allowed-symbols perfect-consonance imperfect-consonance mild-dissonance strong-dissonance neutral)))

;;; Musical context
(deftemplate context
  "Musical context for the sequence"
  (slot detected-key (type STRING) (default "C"))
  (slot detected-mode (type SYMBOL) (allowed-symbols major minor dorian phrygian lydian mixolydian aeolian locrian))
  (slot confidence (type FLOAT) (range 0.0 1.0) (default 0.0))
  (slot tempo (type INTEGER) (default 120))
  (slot ticks-per-quarter (type INTEGER) (default 480)))

;;; Scale membership for a note
(deftemplate scale-membership
  "Whether a note belongs to the detected scale"
  (slot note-index (type INTEGER))
  (slot pitch-class (type SYMBOL))
  (slot in-scale (type SYMBOL) (allowed-symbols yes no)))

;;; Rhythmic entropy summary
(deftemplate rhythmic-entropy
  "Summary of rhythmic variety in the sequence"
  (slot value (type FLOAT) (range 0.0 1.0))
  (slot unique-durations (type INTEGER))
  (slot total-notes (type INTEGER)))

;;; Velocity dynamics summary
(deftemplate dynamics-summary
  "Summary of velocity dynamics"
  (slot min-velocity (type INTEGER))
  (slot max-velocity (type INTEGER))
  (slot velocity-range (type INTEGER))
  (slot velocity-std (type FLOAT)))

;;; Contour information
(deftemplate contour
  "Melodic contour classification"
  (slot type (type SYMBOL) (allowed-symbols ascending descending arch inverse-arch wave static))
  (slot direction-changes (type INTEGER)))

;;; Scoring adjustment (conclusion)
(deftemplate scoring-adjustment
  "Adjustment to a scoring dimension"
  (slot dimension (type SYMBOL) (allowed-symbols harmony melody rhythm resolution dynamics structure))
  (slot amount (type INTEGER) (range -50 50))
  (slot reason (type STRING)))

;;; Improvement suggestion (conclusion)
(deftemplate suggestion
  "Improvement recommendation"
  (slot category (type SYMBOL) (allowed-symbols harmony melody rhythm resolution dynamics structure))
  (slot severity (type SYMBOL) (allowed-symbols info suggestion warning))
  (slot note-index (type INTEGER) (default -1))
  (slot message (type STRING)))

;;; Dissonance tracking
(deftemplate dissonance
  "Track dissonant intervals for resolution analysis"
  (slot interval-index (type INTEGER))
  (slot from-note (type INTEGER))
  (slot to-note (type INTEGER))
  (slot resolved (type SYMBOL) (allowed-symbols yes no pending)))
