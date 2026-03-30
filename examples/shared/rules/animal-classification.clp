;;; Animal Classification Expert System
;;;
;;; A simple educational example demonstrating CLIPS templates and rules.
;;; Classifies animals based on observable characteristics.
;;;
;;; USAGE:
;;;   cargo run --example clips_animal_classification --features clips
;;;
;;; INPUT: Facts with template "animal" containing characteristics
;;; OUTPUT: Facts with template "classification" containing category and reason
;;;
;;; CATEGORIES: mammal, bird, fish, reptile, amphibian, invertebrate,
;;;             monotreme (egg-laying mammal), flying-bird, flightless-bird
;;;
;;; This example uses deftemplate (not COOL/defclass) as required by nxusKit.

;;; ==========================================================================
;;; Template Definitions
;;; ==========================================================================

(deftemplate animal
    "An animal being classified"
    (slot name (type STRING))
    (slot has-backbone (type SYMBOL) (allowed-symbols yes no unknown) (default unknown))
    (slot body-temperature (type SYMBOL) (allowed-symbols warm cold unknown) (default unknown))
    (slot has-feathers (type SYMBOL) (allowed-symbols yes no unknown) (default unknown))
    (slot has-fur (type SYMBOL) (allowed-symbols yes no unknown) (default unknown))
    (slot has-scales (type SYMBOL) (allowed-symbols yes no unknown) (default unknown))
    (slot lives-in-water (type SYMBOL) (allowed-symbols yes no partial unknown) (default unknown))
    (slot can-fly (type SYMBOL) (allowed-symbols yes no unknown) (default unknown))
    (slot lays-eggs (type SYMBOL) (allowed-symbols yes no unknown) (default unknown)))

(deftemplate classification
    "Classification result for an animal"
    (slot animal-name (type STRING))
    (slot category (type SYMBOL))
    (slot confidence (type SYMBOL) (allowed-symbols high medium low) (default medium))
    (slot reason (type STRING)))

;;; ==========================================================================
;;; Classification Rules
;;; ==========================================================================

;;; Mammal classification
(defrule classify-mammal
    "Classify warm-blooded animals with fur as mammals"
    (animal (name ?n)
            (has-backbone yes)
            (body-temperature warm)
            (has-fur yes)
            (lays-eggs no))
    =>
    (assert (classification
        (animal-name ?n)
        (category mammal)
        (confidence high)
        (reason "Warm-blooded vertebrate with fur that gives live birth"))))

;;; Bird classification
(defrule classify-bird
    "Classify warm-blooded animals with feathers as birds"
    (animal (name ?n)
            (has-backbone yes)
            (body-temperature warm)
            (has-feathers yes))
    =>
    (assert (classification
        (animal-name ?n)
        (category bird)
        (confidence high)
        (reason "Warm-blooded vertebrate with feathers"))))

;;; Fish classification
(defrule classify-fish
    "Classify cold-blooded aquatic animals with scales as fish"
    (animal (name ?n)
            (has-backbone yes)
            (body-temperature cold)
            (has-scales yes)
            (lives-in-water yes))
    =>
    (assert (classification
        (animal-name ?n)
        (category fish)
        (confidence high)
        (reason "Cold-blooded aquatic vertebrate with scales"))))

;;; Reptile classification
(defrule classify-reptile
    "Classify cold-blooded land animals with scales as reptiles"
    (animal (name ?n)
            (has-backbone yes)
            (body-temperature cold)
            (has-scales yes)
            (lives-in-water no))
    =>
    (assert (classification
        (animal-name ?n)
        (category reptile)
        (confidence high)
        (reason "Cold-blooded terrestrial vertebrate with scales"))))

;;; Amphibian classification
(defrule classify-amphibian
    "Classify cold-blooded animals that live partially in water without scales"
    (animal (name ?n)
            (has-backbone yes)
            (body-temperature cold)
            (has-scales no)
            (lives-in-water partial))
    =>
    (assert (classification
        (animal-name ?n)
        (category amphibian)
        (confidence high)
        (reason "Cold-blooded vertebrate with moist skin, lives in water and on land"))))

;;; Invertebrate classification (catch-all for no backbone)
(defrule classify-invertebrate
    "Classify animals without a backbone as invertebrates"
    (animal (name ?n)
            (has-backbone no))
    =>
    (assert (classification
        (animal-name ?n)
        (category invertebrate)
        (confidence medium)
        (reason "Animal without a backbone"))))

;;; Special case: Egg-laying mammal (platypus, echidna)
(defrule classify-monotreme
    "Classify egg-laying mammals as monotremes"
    (animal (name ?n)
            (has-backbone yes)
            (body-temperature warm)
            (has-fur yes)
            (lays-eggs yes))
    =>
    (assert (classification
        (animal-name ?n)
        (category monotreme)
        (confidence high)
        (reason "Rare egg-laying mammal like platypus or echidna"))))

;;; Flying bird subclassification
(defrule subclassify-flying-bird
    "Add flying capability note to birds that can fly"
    (animal (name ?n)
            (has-feathers yes)
            (can-fly yes))
    (classification (animal-name ?n) (category bird))
    =>
    (assert (classification
        (animal-name ?n)
        (category flying-bird)
        (confidence high)
        (reason "Bird capable of flight"))))

;;; Flightless bird subclassification
(defrule subclassify-flightless-bird
    "Identify flightless birds"
    (animal (name ?n)
            (has-feathers yes)
            (can-fly no))
    (classification (animal-name ?n) (category bird))
    =>
    (assert (classification
        (animal-name ?n)
        (category flightless-bird)
        (confidence high)
        (reason "Bird that cannot fly (e.g., penguin, ostrich)"))))
