package dquely_test

const simpleMock = `{
  people(func: type(Person)) {
    uid
    name
    friend
  }
}`

const filterMock = `{
  people(func: type(Person)) @filter(gt(age, 18)) {
    uid
    name
    friends {
      uid
      name
    }
  }
}`

const complexMockAnd = `{
  me(func: has(description))
  @filter(
    ngram(description, "brown fox")
    AND eq(status, "published")
    AND gt(score, 10)
  ) {
    uid
    description
    status
    score
  }
}`

const complexMockOr = `{
  me(func: has(description))
  @filter(
    ngram(description, "brown fox")
    AND (
      eq(status, "published")
      OR eq(status, "review")
    )
  ) {
    uid
    description
    status
  }
}`

const complexFullTextAndDate = `{
  me(func: has(description))
  @filter(
    ngram(description, "brown fox")
    AND ge(created_at, "2025-01-01")
    AND le(created_at, "2025-12-31")
  ) {
    uid
    description
    created_at
  }
}`

const ComplexMockNot = `{
  me(func: has(description))
  @filter(
    ngram(description, "brown fox")
    AND NOT eq(status, "archived")
  ) {
    uid
    description
    status
  }
}`

const AdvancedMock = `{
  me(func: has(description))
  @filter(
    ngram(description, "brown fox")
    AND eq(type, "article")
    AND gt(score, 50)
    AND NOT eq(status, "deleted")
  ) {
    uid
    description
    score
    type
    status
  }
}`

const regexpMock = `{
  directors(func: regexp(name@en, /^Steven Sp.*$/)) {
    name@en
    director.film @filter(regexp(name@en, /ryan/i)) {
      name@en
    }
  }
}`

const alloftextMock = `{
  posts(func: has(description)) @filter(alloftext(description, "quick brown fox")) {
    uid
    description
  }
}`

const anyoftextMock = `{
  posts(func: has(description)) @filter(anyoftext(description, "quick brown fox")) {
    uid
    description
  }
}`

const multiQueryMock = `{
  me(func: has(description))
  @filter(
    ngram(description, "brown fox")
    AND eq(type, "article")
    AND gt(score, 50)
    AND NOT eq(status, "deleted")
  ) {
    uid
    description
    score
    type
    status
  }

  directors(func: regexp(name@en, /^Steven Sp.*$/)) {
    name@en
    director.film @filter(regexp(name@en, /ryan/i)) {
      name@en
    }
  }
}`

const alloftermsMock = `{
  me(func: allofterms(name@en, "jones indiana")) {
    name@en
    genre {
      name@en
    }
  }
}`

const alloftermsFieldMock = `{
  me(func: eq(name@en, "Steven Spielberg")) @filter(has(director.film)) {
    name@en
    director.film @filter(allofterms(name@en, "jones indiana")) {
      name@en
    }
  }
}`

const anyoftermsMock = `{
  me(func: anyofterms(name@en, "poison peacock")) {
    name@en
    genre {
      name@en
    }
  }
}`

const anyoftermsFieldMock = `{
  me(func: eq(name@en, "Steven Spielberg")) @filter(has(director.film)) {
    name@en
    director.film @filter(anyofterms(name@en, "war spies")) {
      name@en
    }
  }
}`

const betweenMock = `{
  me(func: between(initial_release_date, "1977-01-01", "1977-12-31")) {
    name@en
    genre {
      name@en
    }
  }
}`

const simpleUidMock = `{
  films(func: uid(0x2c964)) {
    name@hi
    actor.film {
      performance.film {
        name@hi
      }
    }
  }
}`

const variableUidMock = `{
  var(func: allofterms(name@en, "Taraji Henson")) {
    actor.film {
      F as performance.film {
        G as genre
      }
    }
  }

  Taraji_films_by_genre(func: uid(G)) {
    genre_name : name@en
    films : ~genre @filter(uid(F)) {
      film_name : name@en
    }
  }
}`

const complexVariableAndUidMock = `{
  var(func: allofterms(name@en, "Taraji Henson")) {
    actor.film {
      F as performance.film {
        G as count(genre)
        genre {
          C as count(~genre @filter(uid(F)))
        }
      }
    }
  }

  Taraji_films_by_genre_count(func: uid(G), orderdesc: val(G)) {
    film_name : name@en
    genres : genre (orderdesc: val(C)) {
      genre_name : name@en
    }
  }
}`

const uid_inMock = `{
  caro(func: eq(name@en, "Marc Caro")) {
    name@en
    director.film @filter(uid_in(~director.film, 0x99706)) {
      name@en
    }
  }
}`

const uid_inMultiMock = `{
  caro(func: eq(name@en, "Marc Caro")) {
    name@en
    director.film @filter(uid_in(~director.film, [0x99706,0x99705,0x99704])) {
      name@en
    }
  }
}`

const uid_inWithVarMock = `{
  getJeunet as q(func: eq(name@fr, "Jean-Pierre Jeunet"))

  caro(func: eq(name@en, "Marc Caro")) {
    name@en
    director.film @filter(uid_in(~director.film, uid(getJeunet) )) {
      name@en
    }
  }
}`

// IE(predicate, value) — already covered by existing tests

// IE(val(varName), value)
const gtValKeyMock = `{
  me(func: has(score)) @filter(gt(val(G), 50)) {
    uid
    score
  }
}`

const geValKeyMock = `{
  me(func: has(score)) @filter(ge(val(G), 50)) {
    uid
    score
  }
}`

const leValKeyMock = `{
  me(func: has(score)) @filter(le(val(G), 50)) {
    uid
    score
  }
}`

const ltValKeyMock = `{
  me(func: has(score)) @filter(lt(val(G), 50)) {
    uid
    score
  }
}`

// IE(predicate, val(varName))
const gtValValueMock = `{
  me(func: has(score)) @filter(gt(score, val(G))) {
    uid
    score
  }
}`

const geValValueMock = `{
  me(func: has(score)) @filter(ge(score, val(G))) {
    uid
    score
  }
}`

const leValValueMock = `{
  me(func: has(score)) @filter(le(score, val(G))) {
    uid
    score
  }
}`

const ltValValueMock = `{
  me(func: has(score)) @filter(lt(score, val(G))) {
    uid
    score
  }
}`

// IE(count(predicate), value)
const gtCountMock = `{
  me(func: has(genre)) @filter(gt(count(~genre), 30000)) {
    uid
    name@en
  }
}`

const geCountMock = `{
  me(func: has(genre)) @filter(ge(count(~genre), 30000)) {
    uid
    name@en
  }
}`

const leCountMock = `{
  me(func: has(genre)) @filter(le(count(~genre), 30000)) {
    uid
    name@en
  }
}`

const ltCountMock = `{
  me(func: has(genre)) @filter(lt(count(~genre), 30000)) {
    uid
    name@en
  }
}`

// For positive N, first: N retrieves the first N results, by sorted or UID order.
// For negative N, first: N retrieves the last N results, by sorted or UID order.
// Currently, negative is only supported when no order is applied. To achieve the effect of a negative with a sort,
// reverse the order of the sort and use a positive N.
const queryLimitItems = `{
  me(func: allofterms(name@en, "Steven Spielberg")) {
    director.film(first: -2) {
      name@en
      initial_release_date
      genre(orderasc: name@en, first: 3) {
        name@en
      }
    }
  }
}`

const queryComplexLimitItems = `{
  ID as var(func: allofterms(name@en, "Steven")) @filter(has(director.film)) {
    director.film {
      stars as count(starring)
    }
    totalActors as sum(val(stars))
  }

  mostStars(func: uid(ID), orderdesc: val(totalActors), first: 3) {
    name@en
    stars : val(totalActors)
    director.film {
      name@en
    }
  }
}`

const queryWithOffsetMock = `{
  me(func: allofterms(name@en, "Hark Tsui")) {
    name@zh
    name@en
    director.film(orderasc: name@en, first: 6, offset: 4) {
      genre {
        name@en
      }
      name@zh
      name@en
      initial_release_date
    }
  }
}`

const countMock = `{
  directors(func: gt(count(director.film), 5)) {
    totalDirectors : count(uid)
  }
}`

const countFieldMock = `{
  me(func: allofterms(name@en, "Orlando")) @filter(has(actor.film)) {
    name@en
    count(actor.film)
  }
}`

const countAssignedToValueVariableMock = `{
  var(func: allofterms(name@en, "eat drink man woman")) {
    starring {
      actors as performance.actor {
        totalRoles as count(actor.film)
      }
    }
  }

  edmw(func: uid(actors), orderdesc: val(totalRoles)) {
    name@en
    name@zh
    totalRoles : val(totalRoles)
  }
}`

const orderAscMock = `{
  me(func: allofterms(name@en, "Jean-Pierre Jeunet")) {
    name@fr
    director.film(orderasc: initial_release_date) {
      name@fr
      name@en
      initial_release_date
    }
  }
}`

const orderComplexMock = `{
  genres as var(func: has(~genre)) {
    ~genre {
      numGenres as count(genre)
    }
  }

  genres(func: uid(genres), orderasc: name@en) {
    name@en
    ~genre(orderdesc: val(numGenres), first: 5) {
      name@en
      genres : val(numGenres)
    }
  }
}`
