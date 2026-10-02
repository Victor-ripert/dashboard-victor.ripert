const services = [
  { name: "Gateway", latency: "12 ms", status: "Online" },
  { name: "Trading API", latency: "48 ms", status: "Online" },
  { name: "Weather API", latency: "91 ms", status: "Online" },
  { name: "GitHub API", latency: "—", status: "TODO" },
];

export default function HealthWidget() {
  return (
    <div className="flex h-full flex-col justify-around space-y-1 text-xs">
      {services.map((service) => (
        <div key={service.name} className="flex h-7 items-center rounded-md px-2 hover:bg-white/[0.025]">
          <span className={`size-1.5 rounded-full ${service.status === "Online" ? "bg-emerald-400" : "bg-amber-400"}`} />
          <span className="ml-2 font-medium text-slate-300">{service.name}</span>
          <span className="mono ml-auto text-[11px] text-slate-500">{service.latency}</span>
          <span className={`ml-2.5 text-[11px] ${service.status === "Online" ? "text-emerald-400" : "text-amber-400"}`}>
            {service.status}
          </span>
        </div>
      ))}
    </div>
  );
}
