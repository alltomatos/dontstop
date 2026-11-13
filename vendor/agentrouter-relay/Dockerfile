FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY relay.py ./

RUN useradd --system --uid 10001 relay \
    && chown -R relay:relay /app
USER relay

ENV AGENTROUTER_PORT=7187 \
    AGENTROUTER_BIND=0.0.0.0 \
    AGENTROUTER_UPSTREAM=https://agentrouter.org

EXPOSE 7187

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:7187/health', timeout=3)" || exit 1

CMD ["python", "relay.py"]
