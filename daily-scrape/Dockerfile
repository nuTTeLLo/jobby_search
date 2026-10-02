# syntax=docker/dockerfile:1
# The syntax line enables ADD from a git URL (BuildKit), used below for JobSpy.

# 3.10 to match the MCP image: JobSpy pins numpy==1.26.3, which has no wheels for
# newer Pythons.
FROM python:3.10-slim

WORKDIR /app

# JobSpy from the jobspy-mcp-server fork, which adds the Seek scraper and apply_type.
# Pinned to a commit so a fork change cannot silently alter the scrape; bump it
# together with the jobspy-mcp-server submodule pointer.
ARG JOBSPY_REF=681e08bb642247fad9f6b71fed403a15a96cc3d3
ADD https://github.com/nuTTeLLo/jobspy-mcp-server.git#${JOBSPY_REF}:jobspy /opt/jobspy
RUN pip install --no-cache-dir -r /opt/jobspy/requirements.txt "pyjwt==2.*"
ENV PYTHONPATH=/opt/jobspy

COPY board_scrape.py config.json ./

# Timezone data so the container's log timestamps read as Melbourne time when TZ
# is set by the CronJob. Scheduling itself is handled by the CronJob's timeZone.
RUN apt-get update \
    && apt-get install -y --no-install-recommends tzdata \
    && rm -rf /var/lib/apt/lists/*

# Runs as a scheduled job with no inbound traffic, but there is no reason for it
# to be root.
RUN useradd --create-home --uid 10001 scraper
USER scraper

ENTRYPOINT ["python", "/app/board_scrape.py"]
