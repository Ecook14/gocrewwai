import axios from 'axios';

const client = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

export const kickoffCrew = async (payload: any) => {
  const response = await client.post('/crews/kickoff', payload);
  return response.data;
};

export const getSession = async (id: string) => {
  const response = await client.get(`/sessions/${id}`);
  return response.data;
};

export default client;
