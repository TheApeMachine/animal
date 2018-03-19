import dialogflow

class Dialog:

    def __init__(self):
        client   = dialogflow.AgentsClient()
        parent   = client.project_path('animal-ai')
        response = client.export_agent(parent)
