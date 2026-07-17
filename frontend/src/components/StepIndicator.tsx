import '../App.css'

export default function StepIndicator({ current }: { current: 1 | 2 }) {
  return (
    <div className="stepIndicator">
      <div className={`step ${current === 1 ? 'active' : ''}`}>
        <span className="stepDot">1</span>
        <span>Conectar</span>
      </div>
      <span className="stepLine" />
      <div className={`step ${current === 2 ? 'active' : ''}`}>
        <span className="stepDot">2</span>
        <span>Disparar</span>
      </div>
    </div>
  )
}
