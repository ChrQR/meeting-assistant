from typing import BinaryIO

import numpy as np
import soundfile as sf
import torch
from speechbrain.inference.speaker import EncoderClassifier


class SpeakerClassifier:
    def __init__(self):
        self.classifier = EncoderClassifier.from_hparams(
            source="speechbrain/spkrec-ecapa-voxceleb", run_opts={"device": "cpu"}
        )

    def get_embedding_from_slice(
        self, file: BinaryIO, start_sec: float, end_sec: float
    ):
        info = sf.info(file)
        start_frame = int(start_sec * info.samplerate)
        frames = int((end_sec - start_sec) * info.samplerate)

        audio, sr = sf.read(
            file, start=start_frame, stop=start_frame + frames, dtype="float32"
        )

        # convert to mono torch tensor & resample to 16kHz
        if audio.ndim > 1:
            audio = audio.mean(axis=1)
        tensor = torch.tensor(audio).unsqueeze(0)

        # compute embedding
        with torch.no_grad():
            embedding = self.classifier.encode_batch(tensor).squeeze().numpy()

        return embedding / np.linalg.norm(embedding)


def get_speaker_classifier() -> SpeakerClassifier:
    classifier = SpeakerClassifier()
    return classifier
