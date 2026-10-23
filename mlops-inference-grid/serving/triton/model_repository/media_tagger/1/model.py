"""Triton Python backend skeleton for media tagging.

Boots the model repository and returns a stable LABEL tensor.
Replace execute() with the real vision model and keep the tensor names.
"""

import json

import numpy as np
import triton_python_backend_utils as pb_utils


class TritonPythonModel:
    def initialize(self, args):
        self.model_config = json.loads(args["model_config"])

    def execute(self, requests):
        responses = []
        for _ in requests:
            label = np.array([b"unspecified"], dtype=np.object_)
            tensor = pb_utils.Tensor("LABEL", label)
            responses.append(pb_utils.InferenceResponse(output_tensors=[tensor]))
        return responses

    def finalize(self):
        pass
