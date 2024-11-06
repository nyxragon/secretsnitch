FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

RUN apt-get update && apt-get install -y \
    wget \
    unzip \
    libnss3 \
    libgconf-2-4 \
    chromium-driver

RUN pip install selenium==4.18.1
RUN pip install jsbeautifier

COPY fetch_page.py /app/fetch_page.py

WORKDIR /app

ENTRYPOINT ["python", "fetch_page.py"]