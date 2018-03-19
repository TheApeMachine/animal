import psycopg2

class DB:

    def __init__(self):
        self.connection = psycopg2.connect("dbname='animal' user='postgres' host='localhost' password='postgres'")

    def setup(self):
        cursor = self.connection.cursor()
        cursor.execute("CREATE TABLE dialogs(value TEXT)")
