"""Run with the repo's venv: ../.venv/bin/python -m unittest test_board_scrape"""

import re
import unittest

import board_scrape as scrape

CONFIG = {
    "search_terms": ["Software Engineer"],
    "remote_search_terms": ["Elixir"],
    "worldwide_searches": [{"term": "Elixir", "keywords": ["Elixir Phoenix", "Elixir developer"]}],
}

CARDS = """
<div class="base-search-card" data-entity-urn="urn:li:jobPosting:123">
  <h3 class="base-search-card__title"> Backend Engineer </h3>
  <h4 class="base-search-card__subtitle"> Acme </h4>
  <span class="job-search-card__location"> Berlin, Germany </span>
  <time class="job-search-card__listdate" datetime="2026-11-04">1 day ago</time>
</div>
<div class="base-search-card"><h3 class="base-search-card__title">No id</h3></div>
"""


class PlannedSearches(unittest.TestCase):
    def test_worldwide_runs_on_linkedin_only(self):
        scopes = lambda site: [s.scope for s in scrape.planned_searches(CONFIG, site)]
        self.assertEqual(scopes("seek"), [scrape.SCOPE_LOCAL, scrape.SCOPE_REMOTE])
        self.assertEqual(scopes("linkedin").count(scrape.SCOPE_WORLDWIDE), 2)

    def test_digest_lists_every_search(self):
        labels = [scrape.search_label(s) for s in scrape.planned_searches(CONFIG)]
        self.assertEqual(
            labels,
            [
                "Software Engineer",
                "Elixir (remote)",
                "Elixir Phoenix (worldwide)",
                "Elixir developer (worldwide)",
            ],
        )


class ParseLinkedinCards(unittest.TestCase):
    def test_reads_a_card_and_skips_one_without_an_id(self):
        rows = scrape.parse_linkedin_cards(CARDS)
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["id"], "li-123")
        self.assertEqual(rows[0]["title"], "Backend Engineer")
        self.assertEqual(rows[0]["company"], "Acme")
        self.assertEqual(rows[0]["location"], "Berlin, Germany")
        self.assertEqual(rows[0]["date_posted"], "2026-11-04")
        self.assertEqual(rows[0]["job_url"], "https://www.linkedin.com/jobs/view/123")


class MatchesNiche(unittest.TestCase):
    ROLE = re.compile("(engineer|developer|elixir)", re.I)

    def check(self, title, description):
        fetched = []

        def get_description():
            fetched.append(True)
            return description

        return scrape.matches_niche("Elixir", title, self.ROLE, get_description), bool(fetched)

    def test_title_naming_the_term_needs_no_description(self):
        self.assertEqual(self.check("Senior Elixir Developer", None), (True, False))

    def test_description_naming_the_term_is_enough(self):
        self.assertEqual(self.check("Backend Engineer", "We use Elixir and Phoenix."), (True, True))

    def test_description_not_naming_the_term(self):
        self.assertEqual(self.check("Backend Engineer", "We use Go."), (False, True))

    def test_missing_description(self):
        self.assertEqual(self.check("Backend Engineer", None), (False, True))

    def test_other_jobs_never_cost_a_request(self):
        self.assertEqual(self.check("Treasury Manager", "Elixir"), (False, False))

    def test_similar_company_name_does_not_count(self):
        self.assertEqual(self.check("Support Engineer", "Join Elixirr Digital."), (False, True))


class FetchDetails(unittest.TestCase):
    def test_indeed_is_read_once_from_the_result(self):
        cache = {}
        result = {"apply_type": scrape.APPLY_EASY, "description": "text"}
        first = scrape.fetch_details("indeed", "1", result, [], cache)
        second = scrape.fetch_details("indeed", "1", {}, [], cache)
        self.assertEqual(first, (scrape.APPLY_EASY, "text"))
        self.assertEqual(second, first)


if __name__ == "__main__":
    unittest.main()
