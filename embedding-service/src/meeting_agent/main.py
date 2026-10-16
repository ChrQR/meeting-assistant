import base64
import io
from typing import Annotated

import logfire
from fastapi import Depends, FastAPI
from pydantic import BaseModel, field_validator

from meeting_agent.classifier import SpeakerClassifier, get_speaker_classifier
from meeting_agent.config import get_settings

app = FastAPI()

settings = get_settings()

if settings.logfire_api_key is not None:
    logfire.configure(
        service_name="embedding_service",
        service_version="0.0.1",
        min_level=settings.log_level,
        environment="local-dev",
        token=settings.logfire_api_key,
    )
    logfire.instrument_httpx()
    logfire.instrument_pydantic()
    logfire.instrument_fastapi(app)


class SpeakerData(BaseModel):
    speaker_label: str
    start: float
    end: float


class EmbeddingRequest(BaseModel):
    file: bytes
    speakers: list[SpeakerData]

    @field_validator("file", mode="before")
    @classmethod
    def decode_base64_file(cls, v: bytes | str) -> bytes:
        if isinstance(v, str):
            return base64.b64decode(v)
        # If it arrived as bytes, check if it's base64 encoded text
        try:
            return base64.b64decode(v)
        except Exception:
            return v


class EmbeddingResult(BaseModel):
    speaker_label: str
    embedding: list[float]


class EmbeddingResponse(BaseModel):
    data: list[EmbeddingResult]


@app.post("/embedding")
async def embedding(
    req: EmbeddingRequest,
    classifier: Annotated[SpeakerClassifier, Depends(get_speaker_classifier)],
) -> EmbeddingResponse:
    waveform = classifier.load_audio(io.BytesIO(req.file))

    result: list[EmbeddingResult] = []
    for speaker in req.speakers:
        emb = classifier.get_embedding_from_tensor_slice(
            waveform, speaker.start, speaker.end
        )
        result.append(
            EmbeddingResult(speaker_label=speaker.speaker_label, embedding=emb)
        )

    return EmbeddingResponse(data=result)
