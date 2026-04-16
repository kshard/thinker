# The Kleisli Category as a Formal Abstraction of AI Agent Behaviour

**Dmitry Kolesnikov** · `github.com/kshard/thinker` · April 2026

---

## Abstract

Large-language-model (LLM) agents are typically described informally: a bot
receives a prompt, calls tools, and produces a reply. This paper gives a
categorical account of that intuition. We show that every agent
behaviour — sequential composition, reflective self-correction, and plan-then-execute 
— can be expressed as a morphism in the **Kleisli category** of
the combined error-and-state monad. The resulting algebra has a single primitive morphism ($\text{ReAct}$),
three bridging concepts ($	ext{Arr}$, $	ext{Arrow}$, $	ext{Eff}$),
and three structural combinators ($\text{Seq}$, $\text{Reflect}$,
$\text{ThinkReAct}$) that correspond directly to well-known patterns in
multi-agent system design.


## 1. Introduction

The practical engineering of LLM-backed systems has outpaced its theoretical
foundations. Frameworks accumulate ad-hoc combinators — chains, retries,
map-reduce sweeps — without a common algebraic language that would explain
why some compositions are valid and others are not, or predict how behaviours
compose.

Category theory has long served as the mathematics of composition. The Kleisli
construction, in particular, provides a canonical way to reason about effectful
programs: it turns a monad into a category whose morphisms are *programs with
effects*, and in which composition is exactly monadic bind ($\gg\!=\!$). This
paper applies that machinery to LLM agents operating over a shared mutable
*blackboard* — a typed state record $S$ that accumulates knowledge base.

The central claim is:

> **An LLM agent step is a Kleisli endomorphism $S \twoheadrightarrow S$ on
> the blackboard. Agent composition is Kleisli composition. All standard
> multi-agent patterns are instances of this single algebraic structure.**


## 2. Background

### 2.1 The Error Monad with Context

Let $\text{Ctx}$ be the abstract execution context. Define the base monad:

$$M\,A \;\triangleq\; \text{Ctx} \to \text{Either}(\varepsilon,\, A)$$

This is the $\text{IO}$-monad restricted to errors, with a $\text{Reader}$
component for $\text{Ctx}$. Monadic return and bind are:

$$\eta_A(a) \;\triangleq\; \lambda \gamma.\; \text{Right}(a)$$

$$(m \mathbin{\gg\!=} k) \;\triangleq\; \lambda \gamma.\; \text{case}\; m(\gamma) \;\text{of}\; \text{Left}(e) \Rightarrow \text{Left}(e) \mid \text{Right}(a) \Rightarrow k(a)(\gamma)$$

Bind sequences two computations, short-circuiting on the first error.

### 2.2 The State Monad Transformer

Lifting $M$ with a state component $S$ gives:

$$\text{StateT}\,S\,M\,A \;=\; S \to M\,(A \times S)$$

A value of type $\text{StateT}\,S\,M\,A$ is a program that reads a blackboard
$S$, may fail, and returns both a result of type $A$ and an updated blackboard.
$\text{modify}$, $\text{get}$, and $\text{put}$ are the standard primitives.

### 2.3 The Kleisli Category

Given a monad $(M, \eta, \gg\!=)$, the **Kleisli category** $\mathbf{Kl}(M)$
has:
- **objects**: types
- **morphisms** $A \twoheadrightarrow B$: functions $A \to M\,B$
- **identity** on $A$: $\eta_A : A \to M\,A$
- **composition** $(f : A \twoheadrightarrow B) \mathbin{>\!\!=\!\!>} (g : B \twoheadrightarrow C) : A \twoheadrightarrow C$:

$$x \;\mapsto\; f(x) \mathbin{\gg\!=} g$$

Composition is associative and respects the identity laws — this is exactly
what the monad laws guarantee.

### 2.4 Lenses as Bidirectional Accessors

A **lens** from $S$ to $A$ is a pair $(\text{get}, \text{put})$ where
$\text{get} : S \to A$ extracts a component and
$\text{put} : S \times A \to S$ replaces it [4, 5]. The $\text{put}$ half
is the algebraic structure that this paper requires: it makes $S$ a carrier
for the action of $A$, folding an agent's output back into the blackboard.
Curried, $\text{put}(a) : S \to S$ is the action of command $a$ on $S$.

When $S$ is a product type (a record), the lens for each field are derived
automatically — ensuring that the state monad transformer is consistent
with the blackboard's layout.


## 3. The Agent Algebra

### 3.1 The Bot Morphism

The primitive abstraction is the **Bot** — a morphism in $\mathbf{Kl}(M)$:

$$\text{Bot}[S, A] \;\triangleq\; S \to M\,A$$

It reads the current blackboard $S$, performs LLM reasoning (including all
possible side-effectes), and returns a typed result $A$. Two type parameters are necessary:
the input type $S$ (the blackboard) and the output type $A$ (the result, which
is generally different from $S$).

### 3.2 ReAct: The Primitive Realisation of Bot

The definition of $\text{Bot}[S, A]$ is abstract — it specifies the type of a
Kleisli morphism without prescribing how the LLM interaction is carried out.
The **ReAct** (Reason-and-Act) pattern provides the canonical computational
realisation.

A $\text{ReAct}$ agent encapsulates a cyclic process of *thinking* (LLM
reasoning), *acting* (tool invocation — side effect), and *observing* (feeding tool results
back into the context) that repeats until the model produces a final answer.
Internally, the agent threads a *memory* — an ordered sequence of
observations — and an *encoder/decoder* pair that mediates between the typed
input $A$ / output $B$ and the LLM's textual protocol. Externally, the entire
cycle is collapsed into a single invocation:

$$\text{ReAct}[A, B] \;:\; \text{Bot}[A, B]$$

The internal loop is defined as a least fixed point over the LLM reply stage.
Let $\text{Mem}$ be the memory state, $\text{enc} : A \to M\,\text{Prompt}$
the encoder, $\text{dec} : \text{Reply} \to M\,B$ the decoder,
$\text{llm} : \text{Prompt} \times \text{Mem} \to M\,(\text{Reply} \times \text{Mem})$
the model call, and $\text{tools} : \text{Reply} \to M\,\text{Prompt}$ the
tool-dispatch oracle. Then:

$$\text{ReAct}(a) \;\triangleq\; \text{enc}(a) \mathbin{\gg\!=} \text{loop}$$

$$\text{loop}(p) \;\triangleq\; \text{llm}(p) \mathbin{\gg\!=} \lambda (r, m').\
  \begin{cases}
    \text{dec}(r)                                       & \text{if } r.\text{stage} = \text{RETURN} \\
    \text{tools}(r) \mathbin{\gg\!=} \text{loop}          & \text{if } r.\text{stage} = \text{INVOKE} \\
    \text{Left}(\varepsilon_{\text{aborted}})             & \text{otherwise}
  \end{cases}$$

Because the loop terminates on $\text{RETURN}$ or on error, and external
timeout is enforced by $\text{Ctx}$, $\text{ReAct}$ is a well-defined Kleisli
morphism $A \twoheadrightarrow B$.

**Lemma 3.2** (ReAct is a Bot). *$\text{ReAct}[A, B]$ satisfies
$\text{Bot}[A, B]$: it is a function $A \to M\,B$.*

*Proof.* $\text{enc} : A \to M\,\text{Prompt}$, $\text{loop} : \text{Prompt} \to M\,B$
(by case analysis on the reply stage, each branch returns $M\,B$). Their
Kleisli composition $\text{enc} \mathbin{>\!\!=\!\!>} \text{loop} : A \to M\,B$.
$\square$

The role of $\text{ReAct}$ in the algebra is foundational: it is the only
combinator that actually invokes an LLM. Every other combinator in the
hierarchy ($\text{Arrow}$, $\text{Seq}$, $\text{Reflect}$, $\text{ThinkReAct}$)
is a *structural* composition that takes one or more $\text{Bot}$ values and
produces a new $\text{Bot}$. The $\text{ReAct}$ instances are the leaves of
every composition tree.

### 3.3 From Bot to Arrow: The Bridging Problem

A single $\text{ReAct}$ agent produces a result of type $A$, but the
blackboard has type $S$. To compose agents into a complex task solver we need
*endomorphisms* $S \to M\,S$, not heterogeneous morphisms $S \to M\,A$.
The gap between $\text{Bot}[S, A]$ and $\text{Arr}[S]$ is bridged by
an *effect* — a description of how to fold a bot's output back into the
blackboard and then post-process the result.

**$\text{Eff}[S, A]$** is the composite effect that $\text{Arrow}$ applies.
It has two components:

**$\text{Lens}[S, A]$** is the $\text{put}$ half of a lens (§2.4): it folds
the agent's output $A$ back into the blackboard. Curried, it is
$A \to (S \to S)$, a *left action* of $A$ on $S$. When $S$ is a product type
the lens is derived automatically by structural reflection.

$$\text{Lens}[S, A] \;\triangleq\; S \times A \to S$$

**$\text{Eval}[S]$** is a side-effectful hook that runs *after* the lens has
been applied — persistence, validation, guardrails. Its type coincides with
that of $\text{Arr}[S]$ (both are $S \to M\,S$), but its *role* is different:
$\text{Eval}$ is a single post-processing step, whereas $\text{Arr}$ is the
composite agent that includes the bot call, the lens, and the eval.

$$\text{Eval}[S] \;\triangleq\; S \to M\,S$$

The effect bundles both into a single record. When $\text{Lens}$ is absent
the lens is derived automatically via structural reflection; when
$\text{Eval}$ is absent the identity $\eta_S$ is used.

$$\text{Eff}[S, A] \;\triangleq\; \text{Lens}[S, A] \times \text{Eval}[S]$$

### 3.4 The Kleisli Arrow: $\text{Arr}$ and $\text{Arrow}$

With the bridging types in hand, we define the *Kleisli arrow* — the
endomorphism that every agent step must inhabit — this abstraction enforces determisism for probabilistic $\text{Bot}$ behaviour:

$$\text{Arr}[S] \;\triangleq\; S \to M\,S$$

**Lemma 3.3** (Arr–Bot Isomorphism). *$\text{Arr}[S] \cong \text{Bot}[S, S]$.*

*Proof.* Both are defined as $S \to M\,S$. The isomorphism is the identity
function: given $f : \text{Arr}[S]$, define
$\text{Prompt}(f)(s) \triangleq f(s)$; conversely, given
$b : \text{Bot}[S, S]$, define $f(s) \triangleq b(s)$. $\square$

This isomorphism makes every arrow directly usable wherever a bot is expected —
the algebra is *closed*.

$\text{Arrow}$ is the canonical lift of $\text{Bot}[S, A]$ into $\text{Arr}[S]$.
It consumes the bridging $\text{Eff}$ to absorb the output type $A$:

$$\text{Arrow} : \text{Bot}[S, A] \times \text{Eff}[S, A] \to \text{Arr}[S]$$

$$\text{Arrow}(b, \varphi) \;\triangleq\; \lambda s.\; b(s) \mathbin{\gg\!=} \lambda a.\; \varphi.\text{Eval}\!\bigl(\varphi.\text{Lens}(s,\, a)\bigr)$$

The two steps are:
1. **Lens put** (pure): $\varphi.\text{Lens}(s,\, a)$ folds the bot's output
   into the blackboard — a morphism in the ordinary category of types.
2. **Kleisli composition** (effectful): $\varphi.\text{Eval}(s')$ applies the
   side-effect — a morphism in $\mathbf{Kl}(M)$.

Together they yield a single Kleisli endomorphism. $\text{Arrow}$ is thus a
mapping from the hom-set $\text{Bot}[S, A]$ into
$\text{End}_{\mathbf{Kl}(M)}(S)$, the monoid of endomorphisms on $S$.

**Lemma 3.4** (Arrow Well-Definedness). *For any $b : \text{Bot}[S, A]$ and
$\varphi : \text{Eff}[S, A]$, $\text{Arrow}(b, \varphi)$ is a valid Kleisli
endomorphism $S \twoheadrightarrow S$.*

*Proof.* By the typing of the constituents. $b : S \to M\,A$. Given $s : S$,
$b(s)$ yields $M\,A$. In the success branch, $\varphi.\text{Lens}(s, a) : S$
since $\text{Lens} : S \times A \to S$. Then $\varphi.\text{Eval} : S \to M\,S$
applied to this result yields $M\,S$. Composing via bind:

$$\text{Arrow}(b, \varphi)(s) \;=\; b(s) \mathbin{\gg\!=} \lambda a.\; \varphi.\text{Eval}(\varphi.\text{Lens}(s, a))$$

which has type $M\,S$. Therefore $\text{Arrow}(b, \varphi) : S \to M\,S = \text{Arr}[S]$. $\square$

### 3.5 Lifting Computations: $\text{Lift}$

Not every task solving step involves an LLM call. Pure computations, validation,
or transformation steps that operate directly on the blackboard are plain
functions $S \to M\,S$. The $\text{Lift}$ combinator injects them into the
Kleisli arrow type so they compose uniformly with bot-backed steps:

$$\text{Lift} : \text{Eval}[S] \to \text{Arr}[S]$$

$$\text{Lift}(e) \;\triangleq\; e$$

**Lemma 3.5** (Lift Embedding). *$\text{Lift}$ is a faithful embedding of
$(\text{Eval}[S], \mathbin{>\!\!=\!\!>})$ into $(\text{Arr}[S], \mathbin{>\!\!=\!\!>})$.*

*Proof.* $\text{Lift}$ is the identity on the underlying function type.
Faithfulness (injectivity) is immediate:
$\text{Lift}(e_1) = \text{Lift}(e_2) \implies e_1 = e_2$. Composition is
preserved because $\text{Arr}[S]$ and $\text{Eval}[S]$ share the same
underlying Kleisli composition. $\square$

$\text{Lift}$ is the formal mechanism by which deterministic code
(guardrails, enrichment, persistence) enters the Kleisli algebra on equal
footing with LLM-backed agents.

### 3.6 Output Functor: $\text{Map}$

$$\text{Map} : \text{Bot}[S, A] \times (S \times A \to B) \to \text{Bot}[S, B]$$

$$\text{Map}(b, f) \;\triangleq\; \lambda s.\; b(s) \mathbin{\gg\!=} \lambda a.\; \eta(f(s, a))$$

The function $f : S \times A \to B$ has access to both the original blackboard
and the bot's output, enabling projections. The mapping $\text{Bot}[S, -]$
defines a functor $\mathbf{Type} \to \mathbf{Type}$ with $\text{Map}$ as
its action on morphisms, parameterised over the reader $S$.

**Lemma 3.6** (Map Identity). *Let $\pi_2 : S \times A \to A$ be the second
projection. Then $\text{Map}(b, \pi_2) = b$.*

*Proof.* For any $s : S$:
$\text{Map}(b, \pi_2)(s) = b(s) \mathbin{\gg\!=} \lambda a.\; \eta(\pi_2(s, a)) = b(s) \mathbin{\gg\!=} \lambda a.\; \eta(a) = b(s) \mathbin{\gg\!=} \eta = b(s)$
by the monad right-identity law. $\square$

**Lemma 3.7** (Map Composition). *For $g : S \times A \to B$ and
$f : S \times B \to C$, define the $S$-indexed composition
$(f \bullet g)(s, a) \triangleq f(s,\, g(s, a))$. Then:*

$$\text{Map}(\text{Map}(b, g),\; f) \;=\; \text{Map}(b,\; f \bullet g)$$

*Proof.* For any $s : S$:

$$\text{Map}(\text{Map}(b, g), f)(s) = \text{Map}(b, g)(s) \mathbin{\gg\!=} \lambda c.\; \eta(f(s, c))$$

Expanding the inner term:

$$= \bigl(b(s) \mathbin{\gg\!=} \lambda a.\; \eta(g(s, a))\bigr) \mathbin{\gg\!=} \lambda c.\; \eta(f(s, c))$$

By monad associativity:

$$= b(s) \mathbin{\gg\!=} \lambda a.\; \bigl(\eta(g(s, a)) \mathbin{\gg\!=} \lambda c.\; \eta(f(s, c))\bigr)$$

By left identity ($\eta(x) \mathbin{\gg\!=} k = k(x)$):

$$= b(s) \mathbin{\gg\!=} \lambda a.\; \eta(f(s,\, g(s, a))) = \text{Map}(b,\; f \bullet g)(s) \quad\square$$


## 4. Compositional Patterns

### 4.1 Sequential Composition: $\text{Seq}$

$$\text{Seq} : \text{Arr}[S]^{*} \to \text{Arr}[S]$$

$$\text{Seq}(f_1, f_2, \ldots, f_n) \;\triangleq\; f_1 \mathbin{>\!\!=\!\!>} f_2 \mathbin{>\!\!=\!\!>} \cdots \mathbin{>\!\!=\!\!>} f_n$$

Each step receives the blackboard produced by the previous step; the first
error short-circuits. The identity element is $\eta_S : S \to M\,S$.

The caller composes independently-typed bots by first lifting each with
$\text{Arrow}$ to absorb the output type parameter:

$$\text{Seq}(\text{Arrow}(b_A : \text{Bot}[S, A],\, \varphi_A),\;\text{Arrow}(b_B : \text{Bot}[S, B],\, \varphi_B)) \;:\; \text{Arr}[S]$$

The intermediate types $A$ and $B$ disappear — absorbed into the blackboard by
$\text{Eff}.\text{Lens}$. This is the key advantage of the Kleisli
formulation: it enables *type-safe erasure of intermediate result types*
without losing information.

**State threading diagram:**

$$S_0 \xrightarrow{f_1} S_1 \xrightarrow{f_2} S_2 \xrightarrow{\;\cdots\;} S_n$$

**Lemma 4.1** (Seq Associativity). *For any $f, g, h : \text{Arr}[S]$:*

$$\text{Seq}(f, \text{Seq}(g, h)) \;=\; \text{Seq}(\text{Seq}(f, g), h) \;=\; \text{Seq}(f, g, h)$$

*Proof.* $\text{Seq}$ is defined as iterated Kleisli composition
$(\mathbin{>\!\!=\!\!>})$. Unfolding:

$$\text{Seq}(f, \text{Seq}(g, h))(s) \;=\; f(s) \mathbin{\gg\!=} \bigl(\lambda s_1.\; g(s_1) \mathbin{\gg\!=} h\bigr)$$

$$\text{Seq}(\text{Seq}(f, g), h)(s) \;=\; \bigl(f(s) \mathbin{\gg\!=} g\bigr) \mathbin{\gg\!=} h$$

By the **monad associativity law**,
$(m \mathbin{\gg\!=} f) \mathbin{\gg\!=} g = m \mathbin{\gg\!=} (\lambda x.\; f(x) \mathbin{\gg\!=} g)$,
these two expressions are identical. $\square$

**Lemma 4.2** (Seq Identity). *Let $\text{id}_S \triangleq \eta_S$. Then for
any $f : \text{Arr}[S]$:*

$$\text{Seq}(\text{id}_S, f) \;=\; f \;=\; \text{Seq}(f, \text{id}_S)$$

*Proof.*
Left identity: $\text{Seq}(\eta_S, f)(s) = \eta_S(s) \mathbin{\gg\!=} f = f(s)$
by the monad left-identity law ($\eta(a) \mathbin{\gg\!=} k = k(a)$).

Right identity: $\text{Seq}(f, \eta_S)(s) = f(s) \mathbin{\gg\!=} \eta_S = f(s)$
by the monad right-identity law ($m \mathbin{\gg\!=} \eta = m$). $\square$

**Corollary 4.3** (Endomorphism Monoid). *$(\text{Arr}[S], \text{Seq}, \eta_S)$
forms a monoid — the endomorphism monoid
$\text{End}_{\mathbf{Kl}(M)}(S)$.*

**Lemma 4.4** (Arrow–Seq Coherence). *Lifting commutes with sequencing:*

$$\text{Seq}(\text{Arrow}(b_1, \varphi_1),\;\text{Arrow}(b_2, \varphi_2)) \;=\; \text{Arrow}(b_1, \varphi_1) \mathbin{>\!\!=\!\!>} \text{Arrow}(b_2, \varphi_2)$$

*Proof.* Immediate from the definition of $\text{Seq}$ as iterated
$\mathbin{>\!\!=\!\!>}$. $\square$

**Lemma 4.5** (Lift–Seq Homomorphism). *$\text{Lift}$ preserves Kleisli
composition:*

$$\text{Seq}(\text{Lift}(e_1),\;\text{Lift}(e_2)) \;=\; \text{Lift}(e_1 \mathbin{>\!\!=\!\!>} e_2)$$

*Proof.* By Lemma 3.5, $\text{Lift}$ is the identity on the underlying
function, so $\text{Seq}(\text{Lift}(e_1), \text{Lift}(e_2)) = e_1 \mathbin{>\!\!=\!\!>} e_2 = \text{Lift}(e_1 \mathbin{>\!\!=\!\!>} e_2)$. $\square$


### 4.3 Verdict and Judge: $\text{Vote}$ and $\text{Judge}$

Before defining the reflection loop we introduce its decision type.

$$\text{Vote}[S] \;\triangleq\; S \times \{-1,\, 0,\, +1\}$$

A value $(s', v)$ carries the blackboard updated with judge feedback $s'$ and
the decision signal $v$:

| $v$  | Meaning    | Behaviour                                       |
| ---- | ---------- | ----------------------------------------------- |
| $+1$ | **accept** | $s'$ is returned as the final blackboard        |
| $0$  | **revise** | $s'$ (carrying critique) forwarded to corrector |
| $-1$ | **reject** | hard reject; error returned immediately         |

$\text{Judge}$ lifts $\text{Bot}[S, A]$ into $\text{Bot}[S, \text{Vote}[S]]$:

$$\text{Judge} : \text{Bot}[S, A] \times \text{Eff}[\text{Vote}[S],\, A] \to \text{Bot}[S, \text{Vote}[S]]$$

$$\text{Judge}(b, \varphi) \;\triangleq\; \lambda s.\; b(s) \mathbin{\gg\!=} \lambda a.\; \varphi.\text{Eval}\!\bigl(\varphi.\text{Lens}\bigl(\langle s, 0 \rangle,\, a\bigr)\bigr)$$

The default $\varphi.\text{Lens}$ is the composed lens
$\text{Vote}[S].\text{State} \circ \ell_{S,A}$ that writes $A$ into the $S$
field of $\text{Vote}$ and leaves the signal at its zero value. The default
$\varphi.\text{Eval}$ is "always accept" ($v \mapsto +1$).

$\text{Judge}$ separates the concern of *evaluating* content (the raw
bot $b$) from the concern of *deciding* whether to accept (the fold $\varphi$),
allowing the same underlying bot to serve different reflection policies.

### 4.4 Reflective Guard Loop: $\text{Reflect}$

$$\text{Reflect} : \text{Bot}[S, \text{Vote}[S]] \times \text{Arr}[S] \times \mathbb{N} \to \text{Bot}[S, S]$$

The reflection pattern implements a bounded guard/review loop. Formally:

$$\text{Reflect}(j, c, n)(s_0) \;\triangleq\; \text{iter}(n, s_0)$$

where $\text{iter}$ is defined by structural recursion on the attempt counter:

$$\text{iter}(0, s) \;=\; \text{Left}(\varepsilon_{\text{exhausted}})$$

$$\text{iter}(k, s) \;=\; j(s) \mathbin{\gg\!=} \lambda (s', v).\; \begin{cases} \eta(s') & \text{if } v = +1 \\ \text{Left}(\varepsilon_{\text{rejected}}) & \text{if } v = -1 \\ c(s') \mathbin{\gg\!=} \lambda s''.\; \text{iter}(k{-}1,\, s'') & \text{if } v = 0 \end{cases}$$

The corrector $c : \text{Arr}[S]$ is a full Kleisli arrow — not a raw
$\text{Bot}[S, B]$. This is the key structural property: the corrector's own
$\text{Eff}$ (lens + eval) is encapsulated at construction time via
$\text{Arrow}(b_c, \varphi_c)$. The reflection loop is oblivious to the
corrector's internal output type.

Because $\text{Reflect}$ returns $S$ in $M$, it satisfies $\text{Bot}[S, S]$
and is therefore isomorphic to $\text{Arr}[S]$ by Lemma 3.3. This allows
nesting a reflection loop inside a $\text{Seq}$ or another
$\text{Reflect}$.

**Lemma 4.6** (Reflect Termination). *For any $j$, $c$, and $n \in \mathbb{N}$,
$\text{Reflect}(j, c, n)$ invokes the judge at most $n$ times and the
corrector at most $n - 1$ times.*

*Proof.* By structural induction on $n$. Base case ($n = 0$): no invocation,
immediate error. Inductive step ($n = k + 1$): the judge is invoked once.
On accept or reject the loop terminates (1 judge invocation, 0 corrector
invocations). On revise, the corrector is invoked once and the recurrence
$\text{iter}(k, s'')$ is entered, which by the inductive hypothesis invokes
the judge at most $k$ times and the corrector at most $k - 1$ times. Total:
$1 + k = n$ judge invocations and $1 + (k - 1) = n - 1$ corrector
invocations. $\square$

**Lemma 4.7** (Reflect Convergence). *If $j$ always returns $v = +1$ (accept),
then $\text{Reflect}(j, c, n) = j \mathbin{\gg\!=} \pi_{\text{State}}$ for
all $n \geq 1$, regardless of $c$.*

*Proof.* On the first iteration, $j(s)$ yields $(s', +1)$ and the loop
returns $\eta(s')$. The corrector $c$ is never invoked. $\square$

### 4.5 Plan-and-Execute: $\text{ThinkReAct}$ and $\text{Think}$

#### 4.5.1 The Plan Combinator: $\text{Think}$

A planner typically returns a list of *task descriptors* $[A]$ that has to be elevated into blackboard per-task state $T$ for the reactor to operate. $\text{Think}$ applyies a *transform* function
$\sigma : S \times A \to T$ to each element:

$$\text{Think} : \text{Bot}[S, [A]] \times (S \times A \to T) \to \text{Bot}[S, [T]]$$

$$\text{Think}(b, \sigma) \;\triangleq\; \lambda s.\; b(s) \mathbin{\gg\!=} \lambda [a_1, \ldots, a_n].\; \eta\bigl([\sigma(s, a_1), \ldots, \sigma(s, a_n)]\bigr)$$

The transform function projects the outer blackboard $S$ and each task item $A$
into the inner per-task state $T$.

**Lemma 4.8** (Think Purity). *$\text{Think}(b, \sigma)$ invokes $b$
exactly once and applies $\sigma$ purely — no monadic effects are introduced
beyond those of $b$ itself.*

*Proof.* The outer $\eta$ wraps the mapped list in the monad without effects.
$\sigma$ is a pure function. $\square$

#### 4.5.2 The Plan-and-Execute Combinator: $\text{ThinkReAct}$

$$\text{ThinkReAct} : \text{Bot}[S, [T]] \times \text{Arr}[T] \times (S \times [T] \to S) \to \text{Arr}[S]$$

$\text{ThinkReAct}$ implements the plan-and-execute pattern.
It is structurally *different* from $\text{Seq}$ and $\text{Reflect}$: it
operates across **two Kleisli categories** — the outer $\mathbf{Kl}(M)(S)$
and the inner $\mathbf{Kl}(M)(T)$ — connected by a unfold/fold pair.

The think bot $p : \text{Bot}[S, [T]]$ produces the task list, where each
element is already in the inner per-task state $T$ (use $\text{Think}$ to
transform from $\text{Bot}[S, [A]]$ when the planner's output type differs).
The reactor $r : \text{Arr}[T]$ is a full Kleisli arrow — a composed agent
($\text{Seq}$, $\text{Reflect}$, nested $\text{ThinkReAct}$) in the inner
state space. The gather $\gamma : S \times [T] \to S$ folds the per-task
results back into the outer state.

$$\text{ThinkReAct}(p, r, \gamma)(s) \;\triangleq\; p(s) \mathbin{\gg\!=} \lambda [t_1, \ldots, t_n].\; \eta\!\left(\gamma\!\Big(s,\; \big[r(t_i) \mid i = 1 \ldots n\big]\Big)\right)$$

where each $r(t_i)$ is evaluated independently (sequentially or in parallel)
and the results are collected into a list before applying $\gamma$.

When $\gamma$ is omitted, the default is the auto-derived lens
$\text{Lens}[S, [T]]$, consistent with $\text{Arrow}$'s default behaviour.

The gather fold absorbs the list of per-task results
into the blackboard, making $\text{ThinkReAct}$ a first-class citizen of the
$\text{Arr}[S]$. It composes directly in $\text{Seq}$, nests inside
$\text{Reflect}$, and can even serve as the react arrow of an outer
$\text{ThinkReAct}$.

This is a critical semantic distinction from $\text{Seq}$:

|                 | $\text{Seq}$                            | $\text{ThinkReAct}$                        |
| --------------- | --------------------------------------- | ------------------------------------------ |
| State threading | Forward: $s_{i+1} = f_i(s_i)$           | None: each task starts from $t_i$          |
| State types     | Single: $S$                             | Two: outer $S$, inner $T$                  |
| Reactor type    | $\text{Arr}[S]$                         | $\text{Arr}[T]$ (composable agent)         |
| Result type     | $S$ (single)                            | $S$ (via fold)                             |
| Category        | $\text{End}_{\mathbf{Kl}(M)}(S)$ monoid | Cross-category traversal $S \to [T] \to S$ |

**Lemma 4.9** (ThinkReAct Independence). *In
$\text{ThinkReAct}(p, r, \gamma)(s)$, the computation $r(t_i)$ does not
depend on the result of $r(t_j)$ for $i \neq j$.*

*Proof.* Each task's computation $r(t_i)$ references only its own per-task
state $t_i$. No variable from another task's computation appears in this
expression. $\square$

**Corollary 4.10** (Parallelisability). *The tasks in $\text{ThinkReAct}$ may
be evaluated in any order or in parallel without affecting the result
(up to non-determinism in $M$).*

**Lemma 4.11** (ThinkReAct is Arr). *$\text{ThinkReAct}(p, r, \gamma) :
\text{Arr}[S]$.*

*Proof.* $p : S \to M\,[T]$. The traversal maps each $t_i$ through
$r : T \to M\,T$ and collects into $M\,[T]$. Then $\gamma(s, {-}) : [T] \to S$
is pure, lifted into $M$ via $\eta$. The overall composition
$s \mapsto p(s) \mathbin{\gg\!=} (\text{traverse}\; r) \mathbin{\gg\!=}
\lambda ts.\; \eta(\gamma(s, ts))$
has type $S \to M\,S = \text{Arr}[S]$. $\square$


## 5. The Full Hierarchy

$$\begin{array}{ll}
\text{ReAct}[A, B] & \text{— primitive LLM agent (reason-act-observe loop)} \\
\quad\big\vert & \\
\text{Bot}[S, A] & \text{— Kleisli morphism } S \to M\,A \\
\quad\big\downarrow\;\text{Arrow}(\varphi) & \text{— absorbs } A \text{ via Eff} \\
\text{Arr}[S] & \text{— Kleisli endomorphism } S \twoheadrightarrow S \;\cong\; \text{Bot}[S, S] \\
\quad\big\vert & \\
\quad\vdash\;\text{Seq}(f, g, \ldots) & \text{— monoid composition } f \mathbin{>\!\!=\!\!>} g \mathbin{>\!\!=\!\!>} \cdots \\
\quad\big\vert & \\
\quad\vdash\;\text{Reflect}(\text{judge}, c, n) & \text{— bounded iteration with guard} \\
\quad\big\vert\quad\quad\vdash\;\text{judge} : \text{Bot}[S, \text{Vote}[S]] & \text{— via Judge}(b, \varphi),\; A \text{ absorbed} \\
\quad\big\vert\quad\quad\vdash\;c : \text{Arr}[S] & \text{— corrector (Bot + Eff, } B \text{ absorbed)} \\
\quad\big\vert & \\
\quad\vdash\;\text{ThinkReAct}(p, r, \gamma) & \text{— transform/fold traversal } S \to [T] \to S \\
\quad\quad\quad\;\vdash\;p : \text{Bot}[S, [T]] & \text{— planner (via Think}(\sigma)\text{)} \\
\quad\quad\quad\;\vdash\;r : \text{Arr}[T] & \text{— reactor (composable agent)} \\
\quad\quad\quad\;\vdash\;\gamma : S \times [T] \to S & \text{— gather fold} \\
\\
\text{Lift}(e) & \text{— embed } \text{Eval}[S] \hookrightarrow \text{Arr}[S] \\
\text{Judge}(b, \varphi) & \text{— lift Bot}[S, A] \to \text{Bot}[S, \text{Vote}[S]] \\
\text{Think}(b, \sigma) & \text{— transform Bot}[S, [A]] \to \text{Bot}[S, [T]] \\
\text{Map}(b, f) & \text{— functor on output: Bot}[S, A] \to \text{Bot}[S, B]
\end{array}$$

Every node in the upper hierarchy satisfies $\text{Bot}[S, {-}]$. Composition
can be arbitrarily nested. As an example, an agent with an inner reflection
loop:

$$\text{inner} \;\triangleq\; \text{Arrow}(b_A,\, \varphi_1) \;:\; \text{Arr}[S]$$
$$\text{reflect} \;\triangleq\; \text{Reflect}(\text{Judge}(b_J, \varphi_J),\;\text{inner},\; 3) \;:\; \text{Bot}[S, S] \cong \text{Arr}[S]$$
$$\text{outer} \;\triangleq\; \text{Seq}(\text{reflect},\;\text{Arrow}(b_B, \varphi_2)) \;:\; \text{Arr}[S]$$
$$\text{result} \;\triangleq\; \text{Map}(\text{outer},\; \lambda(s, s').\; \pi_B(s')) \;:\; \text{Bot}[S, B]$$

The intermediate types $A$ and $B$ are absorbed at each $\text{Arrow}$ and
$\text{Judge}$ boundary. No type annotation mentioning a transient
intermediate type appears in the final agent signature.


## 6. Algebraic Laws and Their Engineering Implications

We collect the key algebraic properties. All proofs appeal only to the three
monad laws of $M$.

### 6.1 Associativity of $\text{Seq}$

**Theorem 6.1.** *$(\text{Arr}[S], \text{Seq}, \eta_S)$ is a monoid.*

*Proof.* Associativity is Lemma 4.1. Left and right identity is Lemma 4.2.
$\square$

**Engineering implication**: agentic behaviour can be grouped and re-grouped
arbitrarily. A three-stage agent assembled top-down behaves identically to
one assembled bottom-up. Refactoring cannot silently break composition order.

### 6.2 Identity Erasure

**Corollary 6.2.** *Optional steps (e.g. a disabled guardrail) can be replaced
by $\eta_S$ without changing agent semantics:*

$$\text{Seq}(f,\; \eta_S,\; g) \;=\; \text{Seq}(f, g)$$

*Proof.* By Lemma 4.2 (right identity) applied to $\text{Seq}(f, \eta_S) = f$,
then substitution. $\square$

### 6.3 $\text{Arrow}$ Coherence

**Theorem 6.3.** *$\text{Arrow}$ maps $\text{Bot}[S, A]$ into
$\text{End}_{\mathbf{Kl}(M)}(S)$, and Kleisli composition of arrows is
$\text{Seq}$:*

$$\text{Arrow}(b_1, \varphi_1) \mathbin{>\!\!=\!\!>} \text{Arrow}(b_2, \varphi_2) \;=\; \text{Seq}(\text{Arrow}(b_1, \varphi_1),\; \text{Arrow}(b_2, \varphi_2))$$

*Proof.* By definition, $\text{Seq}$ is iterated $\mathbin{>\!\!=\!\!>}$.
For a two-element sequence the expansion is the single Kleisli composition.
$\square$

The coherence property means that lifting commutes with sequencing — a
desirable condition for any combinator library.

### 6.4 $\text{Map}$ Functor Laws

**Theorem 6.4.** *$\text{Map}$ satisfies the functor laws under $S$-indexed
composition $(\bullet)$:*

1. $\text{Map}(b, \pi_2) = b$ *(identity — Lemma 3.6)*
2. $\text{Map}(\text{Map}(b, g), f) = \text{Map}(b, f \bullet g)$ *(composition — Lemma 3.7)*

These guarantee that output transformations compose predictably and that
$\text{Map}(b, f)$ does not re-execute the bot.

---

## 7. Comparison with Existing Frameworks

| Framework               | Composition primitive         | Type safety             | Algebraic guarantees     |
| ----------------------- | ----------------------------- | ----------------------- | ------------------------ |
| LangChain (Python)      | `chain \|` operator           | Runtime                 | None stated              |
| LlamaIndex              | Pipeline DAG (dict-typed)     | Runtime                 | None stated              |
| AutoGen                 | Conversational loops          | Runtime                 | None stated              |
| DSPy                    | Module composition            | Partial (Python)        | Informal                 |
| **nanobot (this work)** | Kleisli arrow $\text{Arr}[S]$ | Compile-time (generics) | Monad laws, functor laws |

The key differentiator is **compile-time type safety across composition
boundaries**. The intermediate output type $A$ of any bot is *absorbed* into
the blackboard by $\text{Arrow}$ at construction time; the agent's type
signature $\text{Arr}[S]$ does not mention transient intermediate types. No
runtime type assertion is required.


## 8. Discussion

### 8.1 Blackboard as a First-Class Concept

The design choice to make the blackboard type $S$ explicit in every bot
signature — rather than threading implicit context — has algebraic
consequences. It makes $\text{Bot}[S, A]$ a genuine functor from the category
of state types to the category of effectful computations, enabling the
$\text{Map}$ combinator and the automatic lens derivation in $\text{Arrow}$.

### 8.2 Why Three Patterns, Not One

$\text{Seq}$, $\text{Reflect}$, and $\text{ThinkReAct}$ are all built from
the same primitive types, but they are categorically distinct:

- $\text{Seq}$ is a *monoid* product in
  $\text{End}_{\mathbf{Kl}(M)}(S)$.
- $\text{Reflect}$ is a *bounded iteration* with a decision guard in
  $\mathbf{Kl}(M)(S)$.
- $\text{ThinkReAct}$ is a *cross-category traversal* bridging
  $\mathbf{Kl}(M)(S)$ and $\mathbf{Kl}(M)(T)$ via transform/fold.

Both $\text{Reflect}$ and $\text{ThinkReAct}$ share a common structure: a
specialised bot ($\text{Bot}[S, \text{Vote}[S]]$ or $\text{Bot}[S, [T]]$)
followed by an arrow tail ($\text{Arr}[S]$ or $\text{Arr}[T]$). The key
difference is that $\text{ThinkReAct}$ crosses state spaces — the reactor
$\text{Arr}[T]$ operates in the inner per-task category while the overall
combinator produces $\text{Arr}[S]$ via the gather fold.

All three produce $\text{Arr}[S]$ (or the isomorphic $\text{Bot}[S, S]$),
making them freely composable with each other. The $\text{Think}$ combinator
provides the bridge between the planner's output type $[A]$ and the reactor's
state type $T$.

### 8.3 $\text{Reflect}$ as a Bounded Fixed Point

The reflection loop is a bounded iteration of the form:

$$s_{i+1} = c((\pi_{\text{State}} \circ j)(s_i))$$

This is not a general fixed point (which would require a complete lattice and
a monotone function); it is a *bounded unfolding* of the recurrence,
implemented by Kleisli composition inside a structural recursion on the attempt
counter (Lemma 4.6). The bound $n$ is an engineering safeguard against
non-termination due to a misbehaving judge or corrector.

### 8.4 $\text{Vote}$ and $\text{Judge}$: Separation of Evaluation and Decision

The decomposition of the reflection loop's input into two separate concerns —
evaluating content ($\text{Bot}[S, A]$) and rendering a verdict
($\text{Eff}[\text{Vote}[S], A]$) — via the $\text{Judge}$ combinator is a
consequence of the algebra. The raw judge bot need not know about the
$\text{Vote}$ type or the accept/reject protocol. The effect
$\varphi : \text{Eff}[\text{Vote}[S], A]$ encodes the decision policy
separately, promoting reuse: the same underlying evaluator bot can serve
different reflection policies by varying only $\varphi$.

### 8.5 $\text{ReAct}$ as the Sole Source of Non-Determinism

Every combinator in the hierarchy except $\text{ReAct}$ is a *deterministic*
structural transformation: $\text{Arrow}$ applies a pure fold and a
side-effect; $\text{Seq}$ threads state; $\text{Reflect}$ iterates;
$\text{Think}$ scatters; and $\text{ThinkReAct}$ traverses and gathers.
The only source of non-determinism
(and the only component that contacts an external LLM) is $\text{ReAct}$.
This separation has a practical consequence: the structural combinators can
be tested and verified independently of any language model, and the
composition laws (Theorems 6.1–6.4) hold regardless of which model is wired
into the $\text{ReAct}$ leaves.


## 9. Conclusion

We have shown that the behaviour of LLM agents over a shared blackboard is
faithfully modelled by the Kleisli category of the error-state monad. The
three primitive types — $\text{Eval}[S]$ (effectful state transformer),
$\text{Lens}[S, A]$ (lens put), and $\text{Eff}[S, A]$ (their
product) — together with the $\text{Arrow}$ lift and the $\text{Arr}[S]$
type, form a minimal and complete algebra for agent step construction.

The canonical computational realisation of $\text{Bot}$ is the $\text{ReAct}$
pattern — a cyclic reason-act-observe loop that is the sole source of
non-determinism in the algebra. Every other combinator is a deterministic
structural transformation.

The three structural combinators — $\text{Seq}$ (monoid composition),
$\text{Reflect}$ (bounded iteration with $\text{Vote}$/$\text{Judge}$), and
$\text{ThinkReAct}$ (cross-category transform/unfold traversal) — are derivable
instances of standard categorical constructions. $\text{Lift}$,
$\text{Judge}$, $\text{Think}$, and $\text{Map}$ are derived forms that
improve ergonomics without extending the core.

All key properties — associativity of $\text{Seq}$ (Theorem 6.1), identity
erasure (Corollary 6.2), functor laws for $\text{Map}$ (Theorem 6.4), and
termination of $\text{Reflect}$ (Lemma 4.6) — have been proved from the monad
laws alone.


## References

1. Moggi, E. (1991). *Notions of computation and monads.* Information and
   Computation, 93(1), 55–92.
2. Wadler, P. (1995). *Monads for functional programming.* In Advanced
   Functional Programming, LNCS 925.
3. Pierce, B. C. (1991). *Basic Category Theory for Computer Scientists.*
   MIT Press.
4. Foster, J. N., Greenwald, M. B., Moore, J. T., Pierce, B. C., & Schmitt,
   A. (2007). *Combinators for bidirectional tree transformations: A
   linguistic approach to the view-update problem.* ACM TOPLAS, 29(3).
5. Kmett, E. (2012). *lens: Lenses, Folds and Traversals.*
   `hackage.haskell.org/package/lens`.
6. Kolesnikov, D. (2025–2026). *thinker: Type-safe LLM agent framework.*
   `github.com/kshard/thinker`.
