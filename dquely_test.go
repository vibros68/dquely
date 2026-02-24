package dquely_test

import (
	"testing"

	"github.com/vibros68/dquely"
)

func TestSimple(t *testing.T) {
	dql := dquely.NewDQL("people")
	dql = dql.Select("uid", "name", "friend").
		Type("Person")
	var query = dql.Query()
	if query != simpleMock {
		t.Errorf("expected dql.Query() to return %s, got %s", simpleMock, query)
	}
}

func TestFilter(t *testing.T) {
	dql := dquely.NewDQL("people")
	friends := dquely.NewDQL("").Select("uid", "name").As("friends")
	dql = dql.Select("uid", "name", friends).
		Type("Person").Gt("age", 18)
	var query = dql.Query()
	if query != filterMock {
		t.Errorf("expected dql.Query() to return %s, got %s", filterMock, query)
	}
}

func TestComplexAnd(t *testing.T) {
	dql := dquely.NewDQL("me")
	dql = dql.Has("description").
		Ngram("description", "brown fox").
		Eq("status", "published").
		Gt("score", 10).
		Select("uid", "description", "status", "score")
	query := dql.Query()
	if query != complexMockAnd {
		t.Errorf("expected dql.Query() to return %s, got %s", complexMockAnd, query)
	}
}

func TestComplexOr(t *testing.T) {
	dql := dquely.NewDQL("me")
	dql = dql.Has("description").
		Ngram("description", "brown fox").
		Or(dquely.Eq("status", "published"), dquely.Eq("status", "review")).
		Select("uid", "description", "status")
	query := dql.Query()
	if query != complexMockOr {
		t.Errorf("expected dql.Query() to return %s, got %s", complexMockOr, query)
	}
}

func TestComplexFullTextAndDate(t *testing.T) {
	dql := dquely.NewDQL("me")
	dql = dql.Has("description").
		Ngram("description", "brown fox").
		Ge("created_at", "2025-01-01").
		Le("created_at", "2025-12-31").
		Select("uid", "description", "created_at")
	query := dql.Query()
	if query != complexFullTextAndDate {
		t.Errorf("expected dql.Query() to return %s, got %s", complexFullTextAndDate, query)
	}
}

func TestComplexNot(t *testing.T) {
	dql := dquely.NewDQL("me")
	dql = dql.Has("description").
		Ngram("description", "brown fox").
		Not(dquely.Eq("status", "archived")).
		Select("uid", "description", "status")
	query := dql.Query()
	if query != ComplexMockNot {
		t.Errorf("expected dql.Query() to return %s, got %s", ComplexMockNot, query)
	}
}

func TestRegexp(t *testing.T) {
	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.Regexp("name@en", "ryan", "i"))
	dql := dquely.NewDQL("directors").
		Regexp("name@en", "^Steven Sp.*$").
		Select("name@en", film)
	query := dql.Query()
	if query != regexpMock {
		t.Errorf("expected dql.Query() to return %s, got %s", regexpMock, query)
	}
}

func TestAllofterms(t *testing.T) {
	genre := dquely.NewDQL("").Select("name@en").As("genre")
	dql := dquely.NewDQL("me").
		AllOfTerms("name@en", "jones indiana").
		Select("name@en", genre)
	query := dql.Query()
	if query != alloftermsMock {
		t.Errorf("expected dql.Query() to return %s, got %s", alloftermsMock, query)
	}
}

func TestAlloftermsField(t *testing.T) {
	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.AllOfTerms("name@en", "jones indiana"))
	dql := dquely.NewDQL("me").
		Func(dquely.Eq("name@en", "Steven Spielberg")).
		Filter(dquely.Has("director.film")).
		Select("name@en", film)
	query := dql.Query()
	if query != alloftermsFieldMock {
		t.Errorf("expected dql.Query() to return %s, got %s", alloftermsFieldMock, query)
	}
}

func TestAnyofterms(t *testing.T) {
	genre := dquely.NewDQL("").Select("name@en").As("genre")
	dql := dquely.NewDQL("me").
		AnyOfTerms("name@en", "poison peacock").
		Select("name@en", genre)
	query := dql.Query()
	if query != anyoftermsMock {
		t.Errorf("expected dql.Query() to return %s, got %s", anyoftermsMock, query)
	}
}

func TestAnyoftermsField(t *testing.T) {
	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.AnyOfTerms("name@en", "war spies"))
	dql := dquely.NewDQL("me").
		Func(dquely.Eq("name@en", "Steven Spielberg")).
		Filter(dquely.Has("director.film")).
		Select("name@en", film)
	query := dql.Query()
	if query != anyoftermsFieldMock {
		t.Errorf("expected dql.Query() to return %s, got %s", anyoftermsFieldMock, query)
	}
}

func TestUidIn(t *testing.T) {
	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.UidIn("~director.film", "0x99706"))
	dql := dquely.NewDQL("caro").
		Func(dquely.Eq("name@en", "Marc Caro")).
		Select("name@en", film)
	query := dql.Query()
	if query != uid_inMock {
		t.Errorf("expected dql.Query() to return %s, got %s", uid_inMock, query)
	}
}

func TestUidInMulti(t *testing.T) {
	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.UidIn("~director.film", "0x99706", "0x99705", "0x99704"))
	dql := dquely.NewDQL("caro").
		Func(dquely.Eq("name@en", "Marc Caro")).
		Select("name@en", film)
	query := dql.Query()
	if query != uid_inMultiMock {
		t.Errorf("expected dql.Query() to return %s, got %s", uid_inMultiMock, query)
	}
}

func TestUidInWithVar(t *testing.T) {
	condition := dquely.NewCondition("getJeunet", "q").
		Func(dquely.Eq("name@fr", "Jean-Pierre Jeunet"))

	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.UidIn("~director.film", dquely.Uid("getJeunet")))
	dql := dquely.NewDQL("").
		Func(dquely.Eq("name@en", "Marc Caro")).
		Select("name@en", film).
		As("caro")

	query := dquely.Build(condition, dql)
	if query != uid_inWithVarMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", uid_inWithVarMock, query)
	}
}

func TestSimpleUid(t *testing.T) {
	performanceFilm := dquely.NewDQL("").Select("name@hi").As("performance.film")
	actorFilm := dquely.NewDQL("").Select(performanceFilm).As("actor.film")
	dql := dquely.NewDQL("films").
		Uid("0x2c964").
		Select("name@hi", actorFilm)
	query := dql.Query()
	if query != simpleUidMock {
		t.Errorf("expected dql.Query() to return %s, got %s", simpleUidMock, query)
	}
}

func TestVariableUid(t *testing.T) {
	performanceFilm := dquely.NewDQL("").
		Select("G as genre").
		As("performance.film").
		Assign("F")
	actorFilm := dquely.NewDQL("").Select(performanceFilm).As("actor.film")
	q1 := dquely.NewVar().
		AllOfTerms("name@en", "Taraji Henson").
		Select(actorFilm)

	films := dquely.NewDQL("").
		Select("film_name : name@en").
		As("films : ~genre").
		Filter(dquely.Uid("F"))
	q2 := dquely.NewDQL("").
		Uid("G").
		Select("genre_name : name@en", films).
		As("Taraji_films_by_genre")

	query := dquely.Build(q1, q2)
	if query != variableUidMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", variableUidMock, query)
	}
}

func TestComplexVariableAndUid(t *testing.T) {
	genreNested := dquely.NewDQL("").
		Select("C as count(~genre @filter(uid(F)))").
		As("genre")
	performanceFilm := dquely.NewDQL("").
		Select("G as count(genre)", genreNested).
		As("performance.film").
		Assign("F")
	actorFilm := dquely.NewDQL("").Select(performanceFilm).As("actor.film")
	q1 := dquely.NewVar().
		AllOfTerms("name@en", "Taraji Henson").
		Select(actorFilm)

	genresNested := dquely.NewDQL("").
		Select("genre_name : name@en").
		As("genres : genre (orderdesc: val(C))")
	q2 := dquely.NewDQL("").
		Uid("G").
		Order("val(G)", dquely.DESC).
		Select("film_name : name@en", genresNested).
		As("Taraji_films_by_genre_count")

	query := dquely.Build(q1, q2)
	if query != complexVariableAndUidMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", complexVariableAndUidMock, query)
	}
}

func TestBetween(t *testing.T) {
	genre := dquely.NewDQL("").Select("name@en").As("genre")
	dql := dquely.NewDQL("me").
		Between("initial_release_date", "1977-01-01", "1977-12-31").
		Select("name@en", genre)
	query := dql.Query()
	if query != betweenMock {
		t.Errorf("expected dql.Query() to return %s, got %s", betweenMock, query)
	}
}

func TestAlloftext(t *testing.T) {
	dql := dquely.NewDQL("posts").
		Has("description").
		AllOfText("description", "quick brown fox").
		Select("uid", "description")
	query := dql.Query()
	if query != alloftextMock {
		t.Errorf("expected dql.Query() to return %s, got %s", alloftextMock, query)
	}
}

func TestAnyoftext(t *testing.T) {
	dql := dquely.NewDQL("posts").
		Has("description").
		AnyOfText("description", "quick brown fox").
		Select("uid", "description")
	query := dql.Query()
	if query != anyoftextMock {
		t.Errorf("expected dql.Query() to return %s, got %s", anyoftextMock, query)
	}
}

func TestMultiQuery(t *testing.T) {
	q1 := dquely.NewDQL("").
		Has("description").
		Ngram("description", "brown fox").
		Eq("type", "article").
		Gt("score", 50).
		Not(dquely.Eq("status", "deleted")).
		Select("uid", "description", "score", "type", "status").
		As("me")

	film := dquely.NewDQL("").
		Select("name@en").
		As("director.film").
		Filter(dquely.Regexp("name@en", "ryan", "i"))
	q2 := dquely.NewDQL("").
		Regexp("name@en", "^Steven Sp.*$").
		Select("name@en", film).
		As("directors")

	query := dquely.Build(q1, q2)
	if query != multiQueryMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", multiQueryMock, query)
	}
}

func TestAdvanced(t *testing.T) {
	dql := dquely.NewDQL("me")
	dql = dql.Has("description").
		Ngram("description", "brown fox").
		Eq("type", "article").
		Gt("score", 50).
		Not(dquely.Eq("status", "deleted")).
		Select("uid", "description", "score", "type", "status")
	query := dql.Query()
	if query != AdvancedMock {
		t.Errorf("expected dql.Query() to return %s, got %s", AdvancedMock, query)
	}
}

func TestGtValKey(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Gt(dquely.Val("G"), 50).
		Select("uid", "score")
	query := dql.Query()
	if query != gtValKeyMock {
		t.Errorf("expected dql.Query() to return %s, got %s", gtValKeyMock, query)
	}
}

func TestGeValKey(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Ge(dquely.Val("G"), 50).
		Select("uid", "score")
	query := dql.Query()
	if query != geValKeyMock {
		t.Errorf("expected dql.Query() to return %s, got %s", geValKeyMock, query)
	}
}

func TestLeValKey(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Le(dquely.Val("G"), 50).
		Select("uid", "score")
	query := dql.Query()
	if query != leValKeyMock {
		t.Errorf("expected dql.Query() to return %s, got %s", leValKeyMock, query)
	}
}

func TestLtValKey(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Lt(dquely.Val("G"), 50).
		Select("uid", "score")
	query := dql.Query()
	if query != ltValKeyMock {
		t.Errorf("expected dql.Query() to return %s, got %s", ltValKeyMock, query)
	}
}

func TestGtValValue(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Gt("score", dquely.Val("G")).
		Select("uid", "score")
	query := dql.Query()
	if query != gtValValueMock {
		t.Errorf("expected dql.Query() to return %s, got %s", gtValValueMock, query)
	}
}

func TestGeValValue(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Ge("score", dquely.Val("G")).
		Select("uid", "score")
	query := dql.Query()
	if query != geValValueMock {
		t.Errorf("expected dql.Query() to return %s, got %s", geValValueMock, query)
	}
}

func TestLeValValue(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Le("score", dquely.Val("G")).
		Select("uid", "score")
	query := dql.Query()
	if query != leValValueMock {
		t.Errorf("expected dql.Query() to return %s, got %s", leValValueMock, query)
	}
}

func TestLtValValue(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("score").
		Lt("score", dquely.Val("G")).
		Select("uid", "score")
	query := dql.Query()
	if query != ltValValueMock {
		t.Errorf("expected dql.Query() to return %s, got %s", ltValValueMock, query)
	}
}

func TestGtCount(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("genre").
		Gt(dquely.Count("~genre"), 30000).
		Select("uid", "name@en")
	query := dql.Query()
	if query != gtCountMock {
		t.Errorf("expected dql.Query() to return %s, got %s", gtCountMock, query)
	}
}

func TestGeCount(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("genre").
		Ge(dquely.Count("~genre"), 30000).
		Select("uid", "name@en")
	query := dql.Query()
	if query != geCountMock {
		t.Errorf("expected dql.Query() to return %s, got %s", geCountMock, query)
	}
}

func TestLeCount(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("genre").
		Le(dquely.Count("~genre"), 30000).
		Select("uid", "name@en")
	query := dql.Query()
	if query != leCountMock {
		t.Errorf("expected dql.Query() to return %s, got %s", leCountMock, query)
	}
}

func TestLtCount(t *testing.T) {
	dql := dquely.NewDQL("me").
		Has("genre").
		Lt(dquely.Count("~genre"), 30000).
		Select("uid", "name@en")
	query := dql.Query()
	if query != ltCountMock {
		t.Errorf("expected dql.Query() to return %s, got %s", ltCountMock, query)
	}
}

func TestQueryLimitItems(t *testing.T) {
	genre := dquely.NewDQL("").
		Select("name@en").
		As("genre").
		Order("name@en", dquely.ASC).
		First(3)
	directorFilm := dquely.NewDQL("").
		Select("name@en", "initial_release_date", genre).
		As("director.film").
		First(-2)
	dql := dquely.NewDQL("me").
		AllOfTerms("name@en", "Steven Spielberg").
		Select(directorFilm)
	query := dql.Query()
	if query != queryLimitItems {
		t.Errorf("expected dql.Query() to return %s, got %s", queryLimitItems, query)
	}
}

func TestCount(t *testing.T) {
	dql := dquely.NewDQL("directors").
		Func(dquely.Gt(dquely.Count("director.film"), 5)).
		Select("totalDirectors : count(uid)")
	query := dql.Query()
	if query != countMock {
		t.Errorf("expected dql.Query() to return %s, got %s", countMock, query)
	}
}

func TestCountField(t *testing.T) {
	dql := dquely.NewDQL("me").
		AllOfTerms("name@en", "Orlando").
		Filter(dquely.Has("actor.film")).
		Select("name@en", "count(actor.film)")
	query := dql.Query()
	if query != countFieldMock {
		t.Errorf("expected dql.Query() to return %s, got %s", countFieldMock, query)
	}
}

func TestCountAssignedToValueVariable(t *testing.T) {
	performanceActor := dquely.NewDQL("").
		Select("totalRoles as count(actor.film)").
		As("performance.actor").
		Assign("actors")
	starring := dquely.NewDQL("").
		Select(performanceActor).
		As("starring")
	q1 := dquely.NewVar().
		AllOfTerms("name@en", "eat drink man woman").
		Select(starring)

	q2 := dquely.NewDQL("").
		Uid("actors").
		Order("val(totalRoles)", dquely.DESC).
		Select("name@en", "name@zh", "totalRoles : val(totalRoles)").
		As("edmw")

	query := dquely.Build(q1, q2)
	if query != countAssignedToValueVariableMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", countAssignedToValueVariableMock, query)
	}
}

func TestOrder(t *testing.T) {
	directorFilm := dquely.NewDQL("").
		Select("name@fr", "name@en", "initial_release_date").
		As("director.film").
		Order("initial_release_date", dquely.ASC)
	dql := dquely.NewDQL("me").
		AllOfTerms("name@en", "Jean-Pierre Jeunet").
		Select("name@fr", directorFilm)
	query := dql.Query()
	if query != orderAscMock {
		t.Errorf("expected dql.Query() to return %s, got %s", orderAscMock, query)
	}
}

func TestOrderComplex(t *testing.T) {
	nestedGenreInVar := dquely.NewDQL("").
		Select("numGenres as count(genre)").
		As("~genre")
	q1 := dquely.NewVar().
		Has("~genre").
		BlockVar("genres").
		Select(nestedGenreInVar)

	nestedGenreInGenres := dquely.NewDQL("").
		Select("name@en", "genres : val(numGenres)").
		As("~genre").
		Order("val(numGenres)", dquely.DESC).
		First(5)
	q2 := dquely.NewDQL("").
		Uid("genres").
		Order("name@en", dquely.ASC).
		Select("name@en", nestedGenreInGenres).
		As("genres")

	query := dquely.Build(q1, q2)
	if query != orderComplexMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", orderComplexMock, query)
	}
}

func TestGroupByWithComment(t *testing.T) {
	directorFilm := dquely.NewDQL("").
		Select("a as count(uid)", "# a is a genre UID to count value variable").
		As("director.film").
		GroupBy("genre")
	q1 := dquely.NewVar().
		AllOfTerms("name@en", "steven spielberg").
		Select(directorFilm)

	q2 := dquely.NewDQL("").
		Uid("a").
		Order("val(a)", dquely.DESC).
		Select("name@en", "total_movies : val(a)").
		As("byGenre")

	query := dquely.Build(q1, q2)
	if query != groupByWithCommentMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", groupByWithCommentMock, query)
	}
}

func TestCascadeDirective(t *testing.T) {
	performanceCharacter := dquely.NewDQL("").
		Select("name@en").
		As("performance.character")
	performanceActor := dquely.NewDQL("").
		Select("name@en").
		As("performance.actor").
		Filter(dquely.AllOfTerms("name@en", "Warwick"))
	starring := dquely.NewDQL("").
		Select(performanceCharacter, performanceActor).
		As("starring")
	dql := dquely.NewDQL("HP").
		AllOfTerms("name@en", "Harry Potter").
		Cascade().
		Select("name@en", starring)
	query := dql.Query()
	if query != cascadeDirectiveMock {
		t.Errorf("expected dql.Query() to return %s, got %s", cascadeDirectiveMock, query)
	}
}

func TestExpandAll(t *testing.T) {
	q1 := dquely.NewDQL("").
		Has("director.film").
		First(1).
		Select("uid", dquely.ExpandAllBlock("u as uid")).
		As("uids")

	q2 := dquely.NewDQL("").
		Uid("u").
		Select("count(uid)").
		As("result")

	query := dquely.Build(q1, q2)
	if query != expandAllMock {
		t.Errorf("expected dquely.Build() to return %s, got %s", expandAllMock, query)
	}
}

func TestQueryWithOffset(t *testing.T) {
	genre := dquely.NewDQL("").
		Select("name@en").
		As("genre")
	directorFilm := dquely.NewDQL("").
		Select(genre, "name@zh", "name@en", "initial_release_date").
		As("director.film").
		Order("name@en", dquely.ASC).
		First(6).
		Offset(4)
	dql := dquely.NewDQL("me").
		AllOfTerms("name@en", "Hark Tsui").
		Select("name@zh", "name@en", directorFilm)
	query := dql.Query()
	if query != queryWithOffsetMock {
		t.Errorf("expected dql.Query() to return %s, got %s", queryWithOffsetMock, query)
	}
}

func TestQueryComplexLimitItems(t *testing.T) {
	directorFilmInVar := dquely.NewDQL("").
		Select("stars as count(starring)").
		As("director.film")
	q1 := dquely.NewVar().
		AllOfTerms("name@en", "Steven").
		Filter(dquely.Has("director.film")).
		BlockVar("ID").
		Select(directorFilmInVar, "totalActors as sum(val(stars))")

	directorFilmInMost := dquely.NewDQL("").
		Select("name@en").
		As("director.film")
	q2 := dquely.NewDQL("").
		Uid("ID").
		Order("val(totalActors)", dquely.DESC).
		First(3).
		Select("name@en", "stars : val(totalActors)", directorFilmInMost).
		As("mostStars")

	query := dquely.Build(q1, q2)
	if query != queryComplexLimitItems {
		t.Errorf("expected dquely.Build() to return %s, got %s", queryComplexLimitItems, query)
	}
}
