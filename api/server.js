const express = require('express');
const app = express();
app.use(express.json());

// --- Mock handlers — wire to Fabric SDK later ---

app.post('/api/issue', async (req, res) => {
  // TODO: validate body & call Fabric chaincode IssueCreds
  res.json({ ok: true, message: 'IssueCreds called (mock)' });
});

app.post('/api/verify', async (req, res) => {
  // TODO: call VerifyCreds
  res.json({ ok: true, result: { isActive: true, hashMatches: true }, message: 'VerifyCreds called (mock)' });
});

app.post('/api/revoke', async (req, res) => {
  // TODO: call RevokeCreds
  res.json({ ok: true, message: 'RevokeCreds called (mock)' });
});

app.get('/api/audit', async (req, res) => {
  // TODO: call QueryAuditTrail
  res.json({ ok: true, events: [], message: 'QueryAuditTrail called (mock)' });
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`API listening on http://localhost:${PORT}`);
});