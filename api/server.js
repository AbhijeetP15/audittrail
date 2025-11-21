import express from "express";
import crypto from "node:crypto";

const app = express();
app.use(express.json());

// Simple in-memory mock store; swap with Fabric SDK later.
const credentials = new Map(); // credId -> credential doc
const events = [];             // append-only audit log

const required = (body, fields) => {
  const missing = fields.filter((f) => !body[f]);
  if (missing.length) throw new Error(`Missing: ${missing.join(", ")}`);
};

const recordEvent = (credId, holderDid, action, actorId, outcome, reason = "") => {
  const evt = {
    eventId: crypto.randomUUID(),
    credId,
    holderDid,
    action,
    actorId,
    outcome,
    reason,
    occurredAt: new Date().toISOString(),
  };
  events.push(evt);
  return evt;
};

app.post("/api/issue", (req, res) => {
  try {
    required(req.body, ["credId", "holderDid", "credType", "hashedData", "issuerId"]);
    const { credId, holderDid, credType, hashedData, issuerId } = req.body;
    if (credentials.has(credId)) throw new Error("Credential already exists");

    const cred = {
      credId,
      holderDid,
      credType,
      hashedData,
      issuerId,
      status: "Active",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    credentials.set(credId, cred);
    const evt = recordEvent(credId, holderDid, "Issue", issuerId, "Success");
    res.json({ ok: true, credential: cred, event: evt });
  } catch (err) {
    res.status(400).json({ ok: false, error: err.message });
  }
});

app.post("/api/verify", (req, res) => {
  try {
    required(req.body, ["credId", "verifierId"]);
    const { credId, verifierId } = req.body;
    const cred = credentials.get(credId);
    if (!cred) throw new Error("Credential not found");

    const result = {
      credId,
      isActive: cred.status === "Active",
      hashMatches: true, // placeholder until off-chain hash check
      checkedAt: new Date().toISOString(),
    };
    const evt = recordEvent(credId, cred.holderDid, "Verify", verifierId, "Success");
    res.json({ ok: true, result, event: evt });
  } catch (err) {
    res.status(400).json({ ok: false, error: err.message });
  }
});

app.post("/api/revoke", (req, res) => {
  try {
    required(req.body, ["credId", "reason", "revokerId"]);
    const { credId, reason, revokerId } = req.body;
    const cred = credentials.get(credId);
    if (!cred) throw new Error("Credential not found");
    if (cred.status === "Revoked") throw new Error("Already revoked");

    cred.status = "Revoked";
    cred.updatedAt = new Date().toISOString();
    credentials.set(credId, cred);

    const evt = recordEvent(credId, cred.holderDid, "Revoke", revokerId, "Success", reason);
    res.json({ ok: true, credential: cred, event: evt });
  } catch (err) {
    res.status(400).json({ ok: false, error: err.message });
  }
});

app.get("/api/audit", (req, res) => {
  try {
    const holderDid = req.query.holderDid;
    if (!holderDid) throw new Error("holderDid is required");
    const holderEvents = events.filter((e) => e.holderDid === holderDid);
    res.json({ ok: true, events: holderEvents });
  } catch (err) {
    res.status(400).json({ ok: false, error: err.message });
  }
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`API listening on http://localhost:${PORT}`);
});
