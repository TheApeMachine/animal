from __future__ import division
import re
import sys
from google.cloud import speech
from google.cloud.speech import enums
from google.cloud.speech import types
import pyaudio
from six.moves import queue

# Set the sample rate and audio chunk size.
RATE  = 16000
CHUNK = int(RATE / 10)

class MicrophoneStream(object):

    def __init__(self, rate, chunk):
        self._rate  = rate
        self._chunk = chunk
        self._buff  = queue.Queue()
        self.closed = True

    def __enter__(self):
        self._audio_interface = pyaudio.PyAudio()
        self._audio_stream    = self._audio_interface.open(
            format            = pyaudio.paInt16,
            channels          = 1,
            rate              = self._rate,
            input             = True,
            frames_per_buffer = self._chunk,
            stream_callback   = self._fill_buffer
        )

        self.closed = False

        return self

    def __exit__(self, type, value, traceback):
        self._audio_stream.stop_stream()
        self._audio_stream.close()
        self.closed = True
        self._buff.put(None)
        self._audio_interface.terminate()

    def _fill_buffer(self, in_data, frame_count, time_info, status_flags):
        self._buff.put(in_data)
        return None, pyaudio.paContinue

    def generator(self):
        while not self.closed:
            chunk = self._buff.get()

            if chunk is None:
                return

            data = [chunk]

            while True:
                try:
                    chunk = self._buff.get(block=False)

                    if chunk is None:
                        return

                    data.append(chunk)
                except queue.Empty:
                    break

            yield b''.join(data)

def process(responses):
    # Reset the number of current recognized characters.
    num_chars_printed = 0

    # Loop over the list of responses
    for response in responses:
        # Continue from beginning of the loop if nothing is found in this
        # response.
        if not response.results:
            continue

        # Take the first response suggestion from the list and store it in
        # result.
        result = response.results[0]

        # If this result is empty, continue from the beginning of the loop.
        if not result.alternatives:
            continue

        # Take the first alternative from the list, and store this in transcript.
        transcript      = result.alternatives[0].transcript
        overwrite_chars = ' ' * (num_chars_printed - len(transcript))

        # If this is not the final result, write what we have so far to the
        # console.
        if not result.is_final:
            sys.stdout.write(transcript + overwrite_chars + '\r')
            sys.stdout.flush()
        # If this is the final result, just print the entire transcript to the
        # console.
        else:
            print(transcript + overwrite_chars)

            # Don't forget to reset the number of current recognized characters
            # before we restart the process.
            num_chars_printed = 0

def main():
    # Which language do you want to use to speak to your robot?
    language_code = 'en-US'

    # Create a client for Google Cloud Speech API.
    client = speech.SpeechClient()

    # Set the audio encoding, sample rate, and language to send to the API.
    config = types.RecognitionConfig(
        encoding          = enums.RecognitionConfig.AudioEncoding.LINEAR16,
        sample_rate_hertz = RATE,
        language_code     = language_code
    )

    # Pass the previous configuration into the streaming service, and by
    # setting interim_results to True we get almost real-time transcription.
    streaming_config = types.StreamingRecognitionConfig(
        config          = config,
        interim_results = True
    )

    # Start streaming from the microphone.
    with MicrophoneStream(RATE, CHUNK) as stream:
        # Generate an audio clip in memory to send to the API, without creating
        # a file on the hard drive.
        audio_generator = stream.generator()

        # Formulate the requests we need to send to the API from the audio
        # data we have generated.
        requests = (
            types.StreamingRecognizeRequest(audio_content=content)
            for content in audio_generator
        )

        # Collect the responses sent back to us from the Speech API.
        responses = client.streaming_recognize(streaming_config, requests)

        # Send the responses to our process function.
        process(responses)

if __name__ == '__main__':
    main()
