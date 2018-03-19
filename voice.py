import os
from gtts import gTTS

class Voice:

    def __init__(self):
        pass

    def say(self, msg):
        tts = gTTS(text=msg, lang='en', slow=False)
        tts.save("msg.mp3")
        os.system('play msg.mp3 -q')
