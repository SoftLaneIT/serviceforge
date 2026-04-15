/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 * You may obtain a copy of the LICENSE at
 *
 * https://softlaneit.com/LICENSE.txt
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the LICENSE is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the LICENSE for the
 * specific language governing permissions and limitations
 * under the LICENSE.
 */

const modules = [
  { name: "Booking", state: "Enabled", config: "slot_duration, cancellation_policy" },
  { name: "Payment", state: "Planned", config: "stripe_key, tax_mode" },
  { name: "Queue", state: "Planned", config: "discipline, max_size" },
  { name: "Logging", state: "Planned", config: "retention_days, alert_rules" },
];

const phases = [
  { phase: "1", months: "1-3", focus: "Foundation" },
  { phase: "2", months: "3-5", focus: "Core Modules" },
  { phase: "3", months: "5-7", focus: "Developer Experience" },
  { phase: "4", months: "7-9", focus: "Scale and Monetization" },
  { phase: "5", months: "9-12", focus: "Ecosystem" },
];

export default function HomePage() {
  return (
    <main className="page">
      <section className="hero">
        <span className="badge">ServiceForge Platform</span>
        <h1>Tenant-aware configuration-first service platform</h1>
        <p>
          This shell implements the management entry point for API gateway, tenant, auth,
          configuration, and booking. Add modules without changing tenant integration patterns.
        </p>
      </section>

      <section className="grid">
        <article className="card">
          <h3>Active Tenants</h3>
          <p className="metric">24</p>
          <p>Row-level tenant isolation in PostgreSQL (target architecture)</p>
        </article>
        <article className="card">
          <h3>Requests / min</h3>
          <p className="metric">2,310</p>
          <p>Gateway-level metering and throttling hooks are prepared.</p>
        </article>
        <article className="card">
          <h3>Module Events</h3>
          <p className="metric">52,884</p>
          <p>Kafka event flow for booking, payment, queue, and observability events.</p>
        </article>
      </section>

      <section className="card" style={{ marginTop: 18 }}>
        <h2>Service Catalog</h2>
        <table className="table">
          <thead>
            <tr>
              <th>Module</th>
              <th>Status</th>
              <th>Config Surface</th>
            </tr>
          </thead>
          <tbody>
            {modules.map((module) => (
              <tr key={module.name}>
                <td>{module.name}</td>
                <td>
                  <span className="pill">{module.state}</span>
                </td>
                <td>{module.config}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="card" style={{ marginTop: 18 }}>
        <h2>Delivery Roadmap</h2>
        <table className="table">
          <thead>
            <tr>
              <th>Phase</th>
              <th>Timeline</th>
              <th>Primary Focus</th>
            </tr>
          </thead>
          <tbody>
            {phases.map((phase) => (
              <tr key={phase.phase}>
                <td>{phase.phase}</td>
                <td>{phase.months}</td>
                <td>{phase.focus}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </main>
  );
}
