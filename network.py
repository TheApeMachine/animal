import socket
import threading

class Network:

    def __init__(self):
        self.clients = []
        self.lock    = threading.Lock()

    def server(self):
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.bind(127.0.0.1, 23)
        s.listen(4)
