const API_URL = 'http://localhost:3000/api/chat';

export const sendMessageToBackend = async (message) => {
  try {
    const response = await fetch(API_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message }),
    });
    
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    
    const data = await response.json();
    return data.response;
  } catch (error) {
    console.error('Error sending message to backend:', error);
    throw error;
  }
};
