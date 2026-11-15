FROM golang:1.26-bookworm AS adapter
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/speech-provider-piper ./cmd/speech-provider-piper

FROM python:3.12-slim-bookworm
RUN useradd --system --uid 10001 --create-home speech && \
    python3 -m pip install --no-cache-dir 'piper-tts[http]==1.8.0'
ADD --chown=speech:speech https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx /models/en_US-lessac-medium.onnx
ADD --chown=speech:speech https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json /models/en_US-lessac-medium.onnx.json
RUN echo '5efe09e69902187827af646e1a6e9d269dee769f9877d17b16b1b46eeaaf019f  /models/en_US-lessac-medium.onnx' | sha256sum -c - && \
    echo 'efe19c417bed055f2d69908248c6ba650fa135bc868b0e6abb3da181dab690a0  /models/en_US-lessac-medium.onnx.json' | sha256sum -c -
COPY --from=adapter /out/speech-provider-piper /usr/local/bin/speech-provider-piper
ENV PIPER_DATA_DIR=/models PIPER_VOICE=en_US-lessac-medium PROVIDER_ADDR=:8080
USER speech
EXPOSE 8080
ENTRYPOINT ["speech-provider-piper"]
