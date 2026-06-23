package swarm

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>animal swarm dashboard</title>
<style>
:root { color-scheme: dark; font-family: ui-sans-serif, system-ui, sans-serif; background: #101417; color: #e7ecef; }
body { margin: 0; }
main { max-width: 1180px; margin: 0 auto; padding: 24px; }
h1 { font-size: 24px; margin: 0 0 20px; }
h2 { font-size: 14px; margin: 0 0 12px; color: #b7c3ca; text-transform: uppercase; letter-spacing: 0; }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.panel { border: 1px solid #2e3940; border-radius: 8px; padding: 16px; background: #151b1f; min-width: 0; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th, td { text-align: left; padding: 8px; border-bottom: 1px solid #273139; vertical-align: top; }
th { color: #94a3ad; font-weight: 600; }
.wide { grid-column: 1 / -1; }
.empty { color: #788891; font-size: 13px; }
@media (max-width: 760px) { .grid { grid-template-columns: 1fr; } main { padding: 16px; } }
</style>
</head>
<body>
<main>
<h1>animal swarm dashboard</h1>
<div class="grid">
<section class="panel"><h2>Claims</h2><div id="claims"></div></section>
<section class="panel"><h2>Statuses</h2><div id="statuses"></div></section>
<section class="panel wide"><h2>Tasks</h2><div id="tasks"></div></section>
<section class="panel wide"><h2>Task Claims</h2><div id="taskClaims"></div></section>
<section class="panel"><h2>Signals</h2><div id="signals"></div></section>
<section class="panel"><h2>Metrics</h2><div id="metrics"></div></section>
<section class="panel wide"><h2>Contention</h2><div id="contentions"></div></section>
</div>
</main>
<script>
const cells = value => value === undefined || value === null || value === "" ? "" : String(value);
function table(target, columns, rows) {
  if (!rows || rows.length === 0) {
    document.getElementById(target).innerHTML = '<div class="empty">No records</div>';
    return;
  }
  const head = columns.map(column => '<th>' + column.label + '</th>').join('');
  const body = rows.map(row => '<tr>' + columns.map(column => '<td>' + cells(row[column.key]) + '</td>').join('') + '</tr>').join('');
  document.getElementById(target).innerHTML = '<table><thead><tr>' + head + '</tr></thead><tbody>' + body + '</tbody></table>';
}
function render(snapshot) {
  table('claims', [{key:'Prefix', label:'Prefix'}, {key:'ActorID', label:'Actor'}], snapshot.Claims);
  table('statuses', [{key:'ActorID', label:'Actor'}, {key:'State', label:'State'}], snapshot.Statuses);
  table('tasks', [{key:'id', label:'Task'}, {key:'status', label:'Status'}], (snapshot.Tasks || []).map(task => ({id: task.id, status: task.status && task.status.state})));
  table('taskClaims', [{key:'TaskID', label:'Task'}, {key:'ActorID', label:'Actor'}, {key:'ConfirmAfter', label:'Confirm After'}], snapshot.TaskClaims);
  table('signals', [{key:'Kind', label:'Kind'}, {key:'TaskID', label:'Task'}, {key:'Summary', label:'Summary'}], snapshot.Signals);
  table('metrics', [{key:'Name', label:'Name'}, {key:'TaskID', label:'Task'}, {key:'Score', label:'Score'}], snapshot.Metrics);
  table('contentions', [{key:'ActorID', label:'Actor'}, {key:'Prefix', label:'Prefix'}, {key:'HolderID', label:'Holder'}, {key:'Error', label:'Error'}], snapshot.Contentions);
}
const source = new EventSource('/events');
source.addEventListener('snapshot', event => render(JSON.parse(event.data)));
fetch('/snapshot').then(response => response.json()).then(render);
</script>
</body>
</html>
`
