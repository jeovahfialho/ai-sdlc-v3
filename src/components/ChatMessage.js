import React from 'react';

const ChatMessage = ({ message, isUser }) => (
  <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-4`}>
    <div className={`rounded-lg p-3 max-w-xs lg:max-w-md ${isUser ? 'bg-blue-500 text-white' : 'bg-gray-200 text-gray-800'}`}>
      {message}
    </div>
  </div>
);

export default ChatMessage;
