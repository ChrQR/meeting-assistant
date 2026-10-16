from functools import lru_cache
from typing import BinaryIO

import numpy as np
import torch
import torchaudio
import torchaudio.functional as F
from speechbrain.inference.speaker import EncoderClassifier

TARGET_SAMPLE_RATE = 16000


class SpeakerClassifier:
    def __init__(self):
        self.classifier = EncoderClassifier.from_hparams(
            source="speechbrain/spkrec-ecapa-voxceleb",
            run_opts={"device": "cpu"},
        )

    def load_audio(self, file: BinaryIO) -> torch.Tensor:
        """Decodes the MP3 file once, converts to mono, and resamples to 16 kHz."""
        file.seek(0)
        waveform, sr = torchaudio.load(file, format="mp3")

        # Convert to mono if multi-channel -> shape: [1, time]
        if waveform.shape[0] > 1:
            waveform = waveform.mean(dim=0, keepdim=True)

        # Resample to the target sample rate expected by ECAPA-TDNN
        if sr != TARGET_SAMPLE_RATE:
            waveform = F.resample(waveform, sr, TARGET_SAMPLE_RATE)

        return waveform

    def get_embedding_from_tensor_slice(
        self, waveform: torch.Tensor, start_sec: float, end_sec: float
    ) -> list[float]:
        """Slices the pre-decoded 16 kHz tensor and computes the speaker embedding."""
        start_frame = int(start_sec * TARGET_SAMPLE_RATE)
        end_frame = int(end_sec * TARGET_SAMPLE_RATE)

        sliced_waveform = waveform[:, start_frame:end_frame]

        # Fallback to the full waveform if the slice interval is invalid or empty
        if sliced_waveform.shape[-1] == 0:
            sliced_waveform = waveform

        with torch.no_grad():
            embedding = (
                self.classifier.encode_batch(sliced_waveform).squeeze().cpu().numpy()
            )

        # L2-normalization
        norm = np.linalg.norm(embedding)
        if norm > 0:
            embedding = embedding / norm

        return embedding.tolist()


@lru_cache(maxsize=1)
def get_speaker_classifier() -> SpeakerClassifier:
    return SpeakerClassifier()
