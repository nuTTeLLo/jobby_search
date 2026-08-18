FROM python:3.13-slim

WORKDIR /app

# PyJWT is the only dependency; everything else is stdlib.
RUN pip install --no-cache-dir "pyjwt==2.*"

COPY linkedin_daily_scrape.py config.json ./

# Timezone data so the container's log timestamps read as Melbourne time when TZ
# is set by the CronJob. Scheduling itself is handled by the CronJob's timeZone.
RUN apt-get update \
    && apt-get install -y --no-install-recommends tzdata \
    && rm -rf /var/lib/apt/lists/*

# Runs as a scheduled job with no inbound traffic, but there is no reason for it
# to be root.
RUN useradd --create-home --uid 10001 scraper
USER scraper

ENTRYPOINT ["python", "/app/linkedin_daily_scrape.py"]
