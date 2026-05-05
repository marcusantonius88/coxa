import React, { useState, useEffect } from 'react'
import Card from './components/Card'
import Button from './components/Button'
import Input from './components/Input'
import axios from 'axios'
import './index.css'

function App() {
  const [medications, setMedications] = useState([])
  const [schedules, setSchedules] = useState([])
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    frequency_days: 60,
    pet_id: '',
    owner_id: '',
  })
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetchMedications()
    fetchSchedules()
  }, [])

  const fetchMedications = async () => {
    try {
      const response = await axios.get('http://habit-service:8001/medications')
      setMedications(response.data || [])
    } catch (error) {
      console.error('Erro ao buscar medicamentos:', error)
    }
  }

  const fetchSchedules = async () => {
    try {
      const response = await axios.get('http://scheduler-service:8002/schedules')
      setSchedules(response.data || [])
    } catch (error) {
      console.error('Erro ao buscar agendamentos:', error)
    }
  }

  const handleInputChange = (e) => {
    const { name, value } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: name === 'frequency_days' ? parseInt(value) : value
    }))
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)

    try {
      const formDataToSend = new FormData()
      formDataToSend.append('name', formData.name)
      formDataToSend.append('description', formData.description)
      formDataToSend.append('frequency_days', formData.frequency_days)
      formDataToSend.append('pet_id', formData.pet_id)
      formDataToSend.append('owner_id', formData.owner_id)

      const response = await axios.post('http://habit-service:8001/medications', formDataToSend)
      
      if (response.status === 201) {
        setFormData({
          name: '',
          description: '',
          frequency_days: 60,
          pet_id: '',
          owner_id: '',
        })
        alert('Medicamento criado com sucesso!')
        fetchMedications()
      }
    } catch (error) {
      console.error('Erro ao criar medicamento:', error)
      alert('Erro ao criar medicamento')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="gradient-primary py-8 px-4">
        <div className="max-w-6xl mx-auto">
          <h1 className="text-4xl font-bold text-text-primary">🐶 COXA</h1>
          <p className="text-text-secondary mt-2">Care Orchestration & eXperience for Animals</p>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto p-4 mt-8">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Form Card */}
          <Card title="Cadastrar Medicamento">
            <form onSubmit={handleSubmit} className="space-y-4">
              <Input
                label="Nome do Medicamento"
                name="name"
                value={formData.name}
                onChange={handleInputChange}
                placeholder="Ex: Antiflea"
                required
              />
              <Input
                label="Descrição"
                name="description"
                value={formData.description}
                onChange={handleInputChange}
                placeholder="Descreva o medicamento"
              />
              <Input
                label="Frequência (dias)"
                name="frequency_days"
                type="number"
                value={formData.frequency_days}
                onChange={handleInputChange}
                required
              />
              <Input
                label="ID do Pet"
                name="pet_id"
                value={formData.pet_id}
                onChange={handleInputChange}
                placeholder="ID único do pet"
                required
              />
              <Input
                label="ID do Proprietário"
                name="owner_id"
                value={formData.owner_id}
                onChange={handleInputChange}
                placeholder="ID único do proprietário"
                required
              />
              <Button type="submit" disabled={loading}>
                {loading ? 'Criando...' : 'Criar Medicamento'}
              </Button>
            </form>
          </Card>

          {/* Stats Card */}
          <Card title="Estatísticas">
            <div className="space-y-4">
              <div className="bg-background p-4 rounded-lg">
                <p className="text-text-secondary text-sm">Medicamentos Cadastrados</p>
                <p className="text-3xl font-bold text-primary">{medications.length}</p>
              </div>
              <div className="bg-background p-4 rounded-lg">
                <p className="text-text-secondary text-sm">Agendamentos Ativos</p>
                <p className="text-3xl font-bold text-accent">{schedules.length}</p>
              </div>
              <Button onClick={fetchMedications} variant="secondary">
                Atualizar Dados
              </Button>
            </div>
          </Card>
        </div>

        {/* Medications List */}
        <Card title="Medicamentos Cadastrados" className="mt-8">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-text-secondary">
              <thead className="border-b border-border">
                <tr>
                  <th className="text-left py-3 px-4 text-text-primary">Nome</th>
                  <th className="text-left py-3 px-4 text-text-primary">Frequência</th>
                  <th className="text-left py-3 px-4 text-text-primary">Pet ID</th>
                  <th className="text-left py-3 px-4 text-text-primary">Proprietário</th>
                </tr>
              </thead>
              <tbody>
                {medications.length > 0 ? (
                  medications.map((med) => (
                    <tr key={med.id} className="border-b border-border hover:bg-surface transition">
                      <td className="py-3 px-4 text-text-primary">{med.name}</td>
                      <td className="py-3 px-4">{med.frequency_days} dias</td>
                      <td className="py-3 px-4">{med.pet_id}</td>
                      <td className="py-3 px-4">{med.owner_id}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan="4" className="py-8 px-4 text-center text-text-secondary">
                      Nenhum medicamento cadastrado ainda
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </Card>

        {/* Schedules List */}
        <Card title="Agendamentos" className="mt-8">
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-text-secondary">
              <thead className="border-b border-border">
                <tr>
                  <th className="text-left py-3 px-4 text-text-primary">ID</th>
                  <th className="text-left py-3 px-4 text-text-primary">Próxima Dose</th>
                  <th className="text-left py-3 px-4 text-text-primary">Status</th>
                  <th className="text-left py-3 px-4 text-text-primary">Pet ID</th>
                </tr>
              </thead>
              <tbody>
                {schedules.length > 0 ? (
                  schedules.map((schedule) => (
                    <tr key={schedule.id} className="border-b border-border hover:bg-surface transition">
                      <td className="py-3 px-4 text-text-primary font-mono text-xs">{schedule.id.substring(0, 8)}</td>
                      <td className="py-3 px-4">{new Date(schedule.next_due_date).toLocaleDateString()}</td>
                      <td className="py-3 px-4">
                        <span className={`px-3 py-1 rounded-full text-xs font-semibold ${
                          schedule.status === 'due' ? 'bg-red-500 text-white' : 'bg-green-600 text-white'
                        }`}>
                          {schedule.status}
                        </span>
                      </td>
                      <td className="py-3 px-4">{schedule.pet_id}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan="4" className="py-8 px-4 text-center text-text-secondary">
                      Nenhum agendamento ativo
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </Card>
      </main>

      {/* Footer */}
      <footer className="border-t border-border mt-16 py-8 px-4">
        <div className="max-w-6xl mx-auto text-center text-text-secondary">
          <p>&copy; 2026 COXA - Care Orchestration & eXperience for Animals</p>
        </div>
      </footer>
    </div>
  )
}

export default App
