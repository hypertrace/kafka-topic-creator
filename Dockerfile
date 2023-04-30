FROM python:3.11-slim-bullseye

RUN apt update && apt upgrade -y && rm -rf /var/lib/apt/lists/*

COPY ./src/main/python/requirements.txt /requirements.txt

RUN pip3 install -r /requirements.txt

COPY ./src/main/python/topics.py /app/__main__.py

ENTRYPOINT ["python"]
CMD ["/app"]
