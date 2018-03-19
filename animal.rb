#!/usr/bin/env python

import os

from db import DB
from network import Network
from dialog import Dialog
from eyes import Eyes
from ears import Ears
from voice import Voice

def main():
  print "Animal v0.1b"
  print "Written by: Daniel Owen van Dommelen"

  db      = DB()
  network = Network()
  dialog  = Dialog()
  eyes    = Eyes()
  ears    = Ears()
  voice   = Voice()

  db.setup()
  network.server()

  # eyes.run()
  # ears.run()

  voice.say('Animal online')

if __name__ == "__main__":
  os.system('cls' if os.name == 'nt' else 'clear')
  main()
