#!/usr/bin/env python

import os

from db import DB
from dialog import Dialog
from eyes import Eyes
from ears import Ears

def main():
  print "Animal v0.1b"
  print "Written by: Daniel Owen van Dommelen"

  db     = DB()
  dialog = Dialog()
  eyes   = Eyes()
  ears   = Ears()

  db.setup()

  eyes.run()
  ears.run()

if __name__ == "__main__":
  # os.system('cls' if os.name == 'nt' else 'clear')
  main()
