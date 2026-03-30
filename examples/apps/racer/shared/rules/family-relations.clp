;;;; family-relations.clp - Rules for inferring family relationships
;;;;
;;;; Given parent-child facts, infers: grandparent, sibling, cousin, uncle/aunt

;;; =========================================================
;;; TEMPLATES
;;; =========================================================

(deftemplate person
  "A person in the family tree"
  (slot name (type STRING)))

(deftemplate parent-of
  "Direct parent-child relationship"
  (slot parent (type STRING))
  (slot child (type STRING)))

(deftemplate grandparent-of
  "Grandparent relationship (inferred)"
  (slot grandparent (type STRING))
  (slot grandchild (type STRING)))

(deftemplate sibling
  "Sibling relationship (inferred)"
  (slot person1 (type STRING))
  (slot person2 (type STRING)))

(deftemplate cousin
  "Cousin relationship (inferred)"
  (slot person1 (type STRING))
  (slot person2 (type STRING)))

(deftemplate uncle-aunt-of
  "Uncle or aunt relationship (inferred)"
  (slot uncle-aunt (type STRING))
  (slot niece-nephew (type STRING)))

(deftemplate query-result
  "Result of a relationship query"
  (slot query-type (type SYMBOL))
  (slot person1 (type STRING))
  (slot person2 (type STRING))
  (slot result (type SYMBOL) (default no)
    (allowed-symbols yes no)))

;;; =========================================================
;;; INFERENCE RULES
;;; =========================================================

;;; Rule: Infer grandparent relationship
(defrule infer-grandparent
  "If A is parent of B and B is parent of C, then A is grandparent of C"
  (parent-of (parent ?grandparent) (child ?parent))
  (parent-of (parent ?parent) (child ?grandchild))
  (not (grandparent-of (grandparent ?grandparent) (grandchild ?grandchild)))
  =>
  (assert (grandparent-of (grandparent ?grandparent) (grandchild ?grandchild)))
  (printout t "Inferred: " ?grandparent " is grandparent of " ?grandchild crlf))

;;; Rule: Infer sibling relationship
(defrule infer-sibling
  "If A and B share a parent and A != B, then A and B are siblings"
  (parent-of (parent ?parent) (child ?child1))
  (parent-of (parent ?parent) (child ?child2&:(neq ?child1 ?child2)))
  (test (< (str-compare ?child1 ?child2) 0)) ; Avoid duplicates
  (not (sibling (person1 ?child1) (person2 ?child2)))
  =>
  (assert (sibling (person1 ?child1) (person2 ?child2)))
  (printout t "Inferred: " ?child1 " and " ?child2 " are siblings" crlf))

;;; Rule: Infer uncle/aunt relationship
(defrule infer-uncle-aunt
  "If A is sibling of B and B is parent of C, then A is uncle/aunt of C"
  (or
    (sibling (person1 ?sibling1) (person2 ?sibling2))
    (sibling (person1 ?sibling2) (person2 ?sibling1)))
  (parent-of (parent ?sibling2) (child ?niece-nephew))
  (not (uncle-aunt-of (uncle-aunt ?sibling1) (niece-nephew ?niece-nephew)))
  =>
  (assert (uncle-aunt-of (uncle-aunt ?sibling1) (niece-nephew ?niece-nephew)))
  (printout t "Inferred: " ?sibling1 " is uncle/aunt of " ?niece-nephew crlf))

;;; Rule: Infer cousin relationship
(defrule infer-cousin
  "If A's parent and B's parent are siblings, then A and B are cousins"
  (parent-of (parent ?parent1) (child ?child1))
  (parent-of (parent ?parent2) (child ?child2))
  (or
    (sibling (person1 ?parent1) (person2 ?parent2))
    (sibling (person1 ?parent2) (person2 ?parent1)))
  (test (neq ?child1 ?child2))
  (test (< (str-compare ?child1 ?child2) 0)) ; Avoid duplicates
  (not (cousin (person1 ?child1) (person2 ?child2)))
  =>
  (assert (cousin (person1 ?child1) (person2 ?child2)))
  (printout t "Inferred: " ?child1 " and " ?child2 " are cousins" crlf))

;;; =========================================================
;;; QUERY VALIDATION RULES
;;; =========================================================

(defrule validate-grandparent-query
  "Check if grandparent query is true"
  (query-result (query-type grandparent) (person1 ?gp) (person2 ?gc) (result no))
  (grandparent-of (grandparent ?gp) (grandchild ?gc))
  ?q <- (query-result (query-type grandparent) (person1 ?gp) (person2 ?gc))
  =>
  (modify ?q (result yes)))

(defrule validate-sibling-query
  "Check if sibling query is true"
  (query-result (query-type sibling) (person1 ?p1) (person2 ?p2) (result no))
  (or
    (sibling (person1 ?p1) (person2 ?p2))
    (sibling (person1 ?p2) (person2 ?p1)))
  ?q <- (query-result (query-type sibling) (person1 ?p1) (person2 ?p2))
  =>
  (modify ?q (result yes)))

(defrule validate-cousin-query
  "Check if cousin query is true"
  (query-result (query-type cousin) (person1 ?p1) (person2 ?p2) (result no))
  (or
    (cousin (person1 ?p1) (person2 ?p2))
    (cousin (person1 ?p2) (person2 ?p1)))
  ?q <- (query-result (query-type cousin) (person1 ?p1) (person2 ?p2))
  =>
  (modify ?q (result yes)))

(defrule validate-uncle-aunt-query
  "Check if uncle/aunt query is true"
  (query-result (query-type uncle_or_aunt) (person1 ?ua) (person2 ?nn) (result no))
  (uncle-aunt-of (uncle-aunt ?ua) (niece-nephew ?nn))
  ?q <- (query-result (query-type uncle_or_aunt) (person1 ?ua) (person2 ?nn))
  =>
  (modify ?q (result yes)))

;;; =========================================================
;;; UTILITY FUNCTIONS
;;; =========================================================

(deffunction count-relationships (?type)
  "Count how many relationships of a type exist"
  (length$ (find-all-facts ((?f ?type)) TRUE)))
