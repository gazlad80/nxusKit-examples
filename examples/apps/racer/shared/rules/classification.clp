;;;; classification.clp - Rules for animal classification
;;;;
;;;; Classifies animals based on characteristics into: mammal, bird, reptile,
;;;; amphibian, fish using decision tree logic.

;;; =========================================================
;;; TEMPLATES
;;; =========================================================

(deftemplate animal
  "An animal to classify"
  (slot name (type STRING))
  (slot has-fur (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot lays-eggs (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot can-fly (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot has-feathers (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot is-warm-blooded (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot lives-in-water (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot gives-milk (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot has-scales (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot has-gills (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot has-fins (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot has-moist-skin (type SYMBOL) (default unknown) (allowed-symbols yes no unknown))
  (slot metamorphosis (type SYMBOL) (default unknown) (allowed-symbols yes no unknown)))

(deftemplate classification
  "Classification result for an animal"
  (slot animal-name (type STRING))
  (slot class (type SYMBOL)
    (allowed-symbols mammal bird reptile amphibian fish unknown))
  (slot confidence (type SYMBOL)
    (default high)
    (allowed-symbols high medium low)))

(deftemplate classification-rule-fired
  "Track which classification rules fired for an animal"
  (slot animal-name (type STRING))
  (slot rule-name (type SYMBOL)))

;;; =========================================================
;;; CLASSIFICATION RULES
;;; =========================================================

;;; Priority 1: Mammals (highest priority - gives milk is definitive)
(defrule classify-mammal-gives-milk
  "If animal gives milk, it's a mammal (even if it lays eggs like platypus)"
  (declare (salience 100))
  (animal (name ?name) (gives-milk yes))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class mammal) (confidence high)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name gives-milk)))
  (printout t "Classified " ?name " as MAMMAL (gives milk)" crlf))

;;; Priority 2: Mammals with fur (if milk not confirmed)
(defrule classify-mammal-has-fur
  "If animal has fur and is warm-blooded, it's a mammal"
  (declare (salience 90))
  (animal (name ?name) (has-fur yes) (is-warm-blooded yes))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class mammal) (confidence high)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name has-fur)))
  (printout t "Classified " ?name " as MAMMAL (has fur, warm-blooded)" crlf))

;;; Priority 3: Birds (feathers are definitive)
(defrule classify-bird-has-feathers
  "If animal has feathers, it's a bird"
  (declare (salience 80))
  (animal (name ?name) (has-feathers yes))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class bird) (confidence high)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name has-feathers)))
  (printout t "Classified " ?name " as BIRD (has feathers)" crlf))

;;; Priority 4: Fish (gills and fins are definitive)
(defrule classify-fish-has-gills
  "If animal has gills and fins, it's a fish"
  (declare (salience 70))
  (animal (name ?name) (has-gills yes) (has-fins yes))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class fish) (confidence high)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name has-gills-fins)))
  (printout t "Classified " ?name " as FISH (has gills and fins)" crlf))

;;; Priority 5: Amphibians (moist skin + metamorphosis)
(defrule classify-amphibian
  "If animal has moist skin and undergoes metamorphosis, it's an amphibian"
  (declare (salience 60))
  (animal (name ?name) (has-moist-skin yes) (metamorphosis yes))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class amphibian) (confidence high)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name moist-skin-metamorphosis)))
  (printout t "Classified " ?name " as AMPHIBIAN (moist skin, metamorphosis)" crlf))

;;; Priority 5b: Amphibians (cold-blooded, lives in water, lays eggs, no gills)
(defrule classify-amphibian-cold-water
  "If cold-blooded, lives in water, lays eggs, but no gills - likely amphibian"
  (declare (salience 55))
  (animal (name ?name)
          (is-warm-blooded no)
          (lives-in-water yes)
          (lays-eggs yes)
          (has-gills no))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class amphibian) (confidence medium)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name cold-water-eggs)))
  (printout t "Classified " ?name " as AMPHIBIAN (cold-blooded, water, eggs, no gills)" crlf))

;;; Priority 6: Reptiles (scales, cold-blooded, lays eggs)
(defrule classify-reptile-has-scales
  "If animal has scales, is cold-blooded, and lays eggs, it's a reptile"
  (declare (salience 50))
  (animal (name ?name) (has-scales yes) (is-warm-blooded no) (lays-eggs yes))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class reptile) (confidence high)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name has-scales)))
  (printout t "Classified " ?name " as REPTILE (has scales, cold-blooded, lays eggs)" crlf))

;;; Priority 6b: Reptiles (fallback - cold-blooded, lays eggs, not in water, no feathers)
(defrule classify-reptile-fallback
  "Cold-blooded, lays eggs, not aquatic, no feathers - probably reptile"
  (declare (salience 45))
  (animal (name ?name)
          (is-warm-blooded no)
          (lays-eggs yes)
          (lives-in-water no)
          (has-feathers no)
          (has-gills no))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class reptile) (confidence medium)))
  (assert (classification-rule-fired (animal-name ?name) (rule-name cold-eggs-land)))
  (printout t "Classified " ?name " as REPTILE (cold-blooded, eggs, land)" crlf))

;;; Default: Unknown classification
(defrule classify-unknown
  "Default classification when no specific rules match"
  (declare (salience -100))
  (animal (name ?name))
  (not (classification (animal-name ?name)))
  =>
  (assert (classification (animal-name ?name) (class unknown) (confidence low)))
  (printout t "Could not classify " ?name crlf))

;;; =========================================================
;;; UTILITY RULES
;;; =========================================================

(defrule print-all-classifications
  "Print summary of all classifications when done"
  (declare (salience -200))
  =>
  (printout t crlf "=== Classification Summary ===" crlf)
  (do-for-all-facts ((?c classification)) TRUE
    (printout t ?c:animal-name ": " ?c:class " (confidence: " ?c:confidence ")" crlf)))

;;; =========================================================
;;; UTILITY FUNCTIONS
;;; =========================================================

(deffunction count-by-class (?target-class)
  "Count how many animals are classified as a specific class"
  (length$ (find-all-facts ((?c classification))
    (eq ?c:class ?target-class))))

(deffunction get-classification (?animal-name)
  "Get the classification for a specific animal"
  (bind ?result unknown)
  (do-for-fact ((?c classification)) (eq ?c:animal-name ?animal-name)
    (bind ?result ?c:class))
  ?result)
