import csv
import argparse
import ast
import json
import sys
from jobspy import scrape_jobs

def parse_args():
    parser = argparse.ArgumentParser(description='Scrape jobs from various sites')
    parser.add_argument('--site_name', default="indeed", 
                        help='Comma-separated list of sites to scrape: indeed,linkedin,zip_recruiter,glassdoor,google,bayt,naukri')
    parser.add_argument('--search_term', default="software engineer", 
                        help='Search term for jobs')
    parser.add_argument('--google_search_term', default="software engineer jobs near San Francisco, CA since yesterday", 
                        help='Google specific search term')
    parser.add_argument('--location', default="San Francisco, CA", 
                        help='Location for job search')
    parser.add_argument('--results_wanted', type=int, default=20, 
                        help='Number of results wanted')
    parser.add_argument('--hours_old', type=int, default=72, 
                        help='How many hours old the jobs can be')
    parser.add_argument('--country_indeed', default='USA', 
                        help='Country for Indeed search')
    parser.add_argument('--linkedin_fetch_description', action='store_true', 
                        help='Fetch LinkedIn job descriptions (slower)')
    parser.add_argument('--proxies', 
                        help='Comma-separated list of proxies')
    parser.add_argument('--output', 
                        help='Output file path without extension. If not provided, outputs to stdout')
    parser.add_argument('--format', default="json", choices=['csv', 'json'],
                        help='Output format (csv or json)')
    return parser.parse_args()

args = parse_args()

# Convert comma-separated string to list
site_names = args.site_name.split(',')

# Parse proxies if provided
proxies = args.proxies.split(',') if args.proxies else None

jobs = scrape_jobs(
    site_name=site_names,
    search_term=args.search_term,
    google_search_term=args.google_search_term,
    location=args.location,
    results_wanted=args.results_wanted,
    hours_old=args.hours_old,
    country_indeed=args.country_indeed,
    linkedin_fetch_description=args.linkedin_fetch_description,
    proxies=proxies,
)
# print(f"Found {len(jobs)} jobs", file=sys.stderr)

# Output based on selected format
if args.output:
    # Output to file
    output_file = f"{args.output}.{args.format}"
    if args.format == "csv":
        print(jobs.head(), file=output_file)
        jobs.to_csv(output_file, quoting=csv.QUOTE_NONNUMERIC, escapechar="\\", index=False)
        print(f"Jobs saved to {output_file}")
    elif args.format == "json":
        jobs.to_json(output_file, orient="records", indent=2)
        print(f"Jobs saved to {output_file}")
else:
    # Output to stdout
    if args.format == "csv":
        jobs.to_csv(sys.stdout, quoting=csv.QUOTE_NONNUMERIC, escapechar="\\", index=False)
    elif args.format == "json":
        print(jobs.to_json(orient="records", indent=2))
