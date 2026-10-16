import io
from typing import Annotated, BinaryIO

import logfire
from fastapi import Depends, FastAPI
from pydantic import BaseModel

from meeting_agent.classifier import SpeakerClassifier, get_speaker_classifier
from meeting_agent.config import get_settings

app = FastAPI()

settings = get_settings()

if settings.logfire_api_key is not None:
    logfire.configure(
        service_name="meeting_transcriber",
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
    result: list[EmbeddingResult] = []
    for speaker in req.speakers:
        embedding = classifier.get_embedding_from_slice(
            io.BytesIO(req.file), speaker.start, speaker.end
        )
        result.append(
            EmbeddingResult(speaker_label=speaker.speaker_label, embedding=embedding)
        )
    return EmbeddingResponse(data=result)
