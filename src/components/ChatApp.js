import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Send, Settings, User, Moon, Sun, MessageSquare, HelpCircle, Home, File, Folder, Loader, Download } from 'lucide-react';
import { Card, CardContent } from './ui/card';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

const DownloadButton = ({ projectName }) => {
  const handleDownload = () => {
    const downloadUrl = `http://localhost:8080/downloadProject?project=${encodeURIComponent(projectName)}`;
    window.open(downloadUrl, '_blank');
  };

  return (
    <Button onClick={handleDownload} className="bg-blue-500 hover:bg-blue-600 text-white">
      <Download className="mr-2 h-4 w-4" /> Download Project
    </Button>
  );
};

const JsonCode = ({ json }) => {
    let formattedJson;
    try {
      // Tenta fazer o parse do JSON
      const parsedJson = JSON.parse(json);
      formattedJson = JSON.stringify(parsedJson, null, 2);
    } catch (error) {
      // Se o parsing falhar, exibe o JSON original e uma mensagem de erro
      console.error('Error parsing JSON:', error);
      formattedJson = `Error parsing JSON: ${error.message}\n\nOriginal JSON:\n${json}`;
    }

    return (
      <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto">
        <code>{formattedJson}</code>
      </pre>
    );
  };

  const ChatMessage = ({ message, isUser, step, isDownloadButton, projectName }) => {

      const isJson = typeof message === 'string' && message.trim().startsWith('{') && message.trim().endsWith('}');

      if (isDownloadButton) {
        return (
          <div className="flex justify-start mb-4">
            <div className="rounded-lg p-3 bg-white dark:bg-gray-700 text-gray-800 dark:text-white shadow-md">
              <div className="text-xs mb-1">{`Step ${step}`}</div>
              <p className="mb-2">Your project is ready for download!</p>
              <DownloadButton projectName={projectName} />
            </div>
          </div>
        );
      }

    return (
      <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-4`}>
        <div className={`rounded-lg p-3 max-w-xs lg:max-w-md ${isUser ? 'bg-blue-500 text-white' : 'bg-white dark:bg-gray-700 text-gray-800 dark:text-white'} shadow-md`}>
          <div className="text-xs mb-1">{`Step ${step}`}</div>
          {isJson ? <JsonCode json={message} /> : message}
        </div>
      </div>
    );
  };

  const FileTree = ({ structure, onFileSelect }) => {
    const renderTree = (node, path = '') => {
      if (Array.isArray(node)) {
        // Trata array como arquivos
        return node.map((file) => (
          <div key={`${path}/${file}`} className="flex items-center ml-4 cursor-pointer" onClick={() => onFileSelect(`${path}/${file}`)}>
            <File size={16} className="mr-2" />
            <span>{file}</span>
          </div>
        ));
      }
  
      if (typeof node === 'object' && node !== null) {
        return Object.entries(node).map(([key, value]) => {
          // Verifica se é um arquivo (tem extensão ou é 'Dockerfile')
          const isFile = key.includes('.') || key === 'Dockerfile';
  
          if (isFile) {
            // Tratar como arquivo, mesmo que seja um objeto vazio ou sem conteúdo
            return (
              <div key={key} className="flex items-center ml-4 cursor-pointer" onClick={() => onFileSelect(`${path}/${key}`)}>
                <File size={16} className="mr-2" />
                <span>{key}</span>
              </div>
            );
          } else {
            // Se for um diretório, renderizar como diretório
            return (
              <div key={key}>
                <div className="flex items-center">
                  <Folder size={16} className="mr-2" />
                  <span>{key}</span>
                </div>
                <div className="ml-4">{renderTree(value, path ? `${path}/${key}` : key)}</div>
              </div>
            );
          }
        });
      }
  
      return null;
    };
  
    return <div className="mt-4">{renderTree(structure)}</div>;
  };
  
  


const ChatApp = () => {
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [inputType, setInputType] = useState('text');
  const [isLoading, setIsLoading] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [conversationID, setConversationID] = useState(null);
  const [currentStep, setCurrentStep] = useState(1);
  const [awaitingConfirmation, setAwaitingConfirmation] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('Disconnected');
  const [connectionDetails, setConnectionDetails] = useState('');
  const [projectStructure, setProjectStructure] = useState(null);
  const [fileContents, setFileContents] = useState({});
  const [selectedFile, setSelectedFile] = useState(null);
  const [showChat, setShowChat] = useState(true);
  const [creatingProject, setCreatingProject] = useState(false);
  const messagesEndRef = useRef(null);
  const socketRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const [progress, setProgress] = useState({ percentage: 0, message: '' });
  const [projectName, setProjectName] = useState('');

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(scrollToBottom, [messages]);


  const connectWebSocket = useCallback(() => {
    console.log('Attempting to connect to WebSocket...');
    socketRef.current = new WebSocket('ws://localhost:8080/chat');

    socketRef.current.onopen = () => {
      console.log('WebSocket connected');
      setConnectionStatus('Connected');
    };

    socketRef.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      console.log('Received message:', data);
      handleIncomingMessage(data);
    };

    socketRef.current.onclose = (event) => {
      console.log('WebSocket disconnected:', event);
      setConnectionStatus('Disconnected');
      setConnectionDetails(`Closed: Code ${event.code}, Reason: ${event.reason}`);
      reconnectTimeoutRef.current = setTimeout(connectWebSocket, 5000);
    };

    socketRef.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnectionStatus('Error');
    };
  }, []);

  useEffect(() => {
    connectWebSocket();

    return () => {
      if (socketRef.current) {
        socketRef.current.close();
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [connectWebSocket]);

  const ProgressBar = ({ percentage, message }) => (
    <div className="flex flex-col items-center mt-4 w-full max-w-md mx-auto">
      <div className="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
        <div
          className="bg-blue-600 h-2 rounded-full transition-all duration-500 ease-out"
          style={{ width: `${percentage}%` }}
        ></div>
      </div>
      <p className="text-sm mt-2">{message}</p>
      <p className="text-sm font-bold">{percentage}% Complete</p>
    </div>
  );

  const handleIncomingMessage = (data) => {
    console.log('Handling incoming message:', data);

    switch (data.type) {
      case 'chat_response':
        console.log('Received chat response:', data.content);
        setMessages(prevMessages => {
          // Verifica se a mensagem já existe para evitar duplicação
          if (!prevMessages.some(msg => msg.text === data.content.message && msg.step === data.content.step_number)) {
            console.log('Adding new chat response to messages');
            return [...prevMessages, {
              text: data.content.message,
              isUser: false,
              step: data.content.step_number
            }];
          }
          return prevMessages;
        });
        setCurrentStep(data.content.step_number);
        setAwaitingConfirmation(data.content.requires_confirmation);
        if (!conversationID) {
          console.log('Setting new conversation ID:', data.content.conversation_id);
          setConversationID(data.content.conversation_id);
        }
        if (data.content.step_number === 1 && data.content.requires_confirmation) {
          console.log('Adding confirmation request message');
          setMessages(prevMessages => [
            ...prevMessages,
            { text: "Is this what you had in mind? (YES/NO)", isUser: false, step: data.content.step_number }
          ]);
        }
        break;

      case 'project_structure':
        console.log('Received project structure:', data.content);
        setProjectStructure(data.content);
        break;

      case 'file_content':
        console.log('Received file content:', data.content);
        setFileContents(prevContents => ({
          ...prevContents,
          [data.content.path]: data.content.content
        }));
        break;

      case 'status_update':
        console.log('Status update:', data.content);
        setMessages(prevMessages => {
          // Verifica se a mensagem de status já existe
          if (!prevMessages.some(msg => msg.text === data.content && !msg.isUser)) {
            console.log('Adding status update to messages');
            return [...prevMessages, { text: data.content, isUser: false, step: currentStep }];
          }
          return prevMessages;
        });
        if (data.content === "Generating project files...") {
          console.log('Starting project creation process');
          setCreatingProject(true);
          setProgress({ percentage: 0, message: 'Starting project creation...' });
        }

        break;

      case 'progress_update':
        console.log('Progress update:', data.content);
        setProgress(data.content);
        if (data.content.percentage === 100) {
          console.log('Project creation complete. Setting creatingProject to false');
          setCreatingProject(false);
          // Adiciona o botão de download apenas se não existir
          setMessages(prevMessages => {
            if (!prevMessages.some(msg => msg.isDownloadButton)) {
              return [
                ...prevMessages,
                {
                  text: "Your project is ready for download!",
                  isUser: false,
                  step: currentStep + 1,
                  isDownloadButton: true
                }
              ];
            }
            return prevMessages;
          });
        }
        break;

      case 'error':
        console.error('Error from server:', data.content);
        setMessages(prevMessages => {
          console.log('Adding error message to chat');
          return [...prevMessages, { text: `Error: ${data.content}`, isUser: false, step: currentStep }];
        });
        break;

      default:
        console.warn('Unknown message type:', data.type);
        break;
    }

    setIsLoading(false);
    console.log('Message handling complete. isLoading set to false');
  };

  const sendMessage = (message, isConfirmation = false) => {
    console.log('sendMessage called with:', { message, isConfirmation });

    if (message.trim() === '') {
      console.log('Empty message, returning without sending');
      return;
    }

    // Mostrar o loading imediatamente quando a mensagem é enviada
    setIsLoading(true);
    console.log('isLoading set to true');

    // Verifica se o WebSocket está aberto antes de enviar a mensagem
    if (socketRef.current.readyState !== WebSocket.OPEN) {
      console.error('WebSocket is not connected');
      setMessages(prevMessages => {
        console.log('Adding WebSocket error message to chat');
        return [...prevMessages, { text: 'Error: WebSocket is not connected', isUser: false, step: currentStep }];
      });
      setIsLoading(false);
      console.log('isLoading set to false due to WebSocket error');
      return;
    }

    console.log('Preparing to send message:', message);
    const newMessage = { text: message, isUser: true, step: currentStep };
    setMessages(prevMessages => {
      console.log('Adding user message to chat');
      return [...prevMessages, newMessage];
    });
    setInputMessage('');

    // Caso seja a primeira mensagem ou uma confirmação
    if (isConfirmation && message === 'YES' && currentStep === 1) {
      console.log('User confirmed project creation');
      setCreatingProject(true);
      console.log('creatingProject set to true');
      setProgress({ percentage: 0, message: 'Preparing to create project...' });
      console.log('Initial progress set');
      setMessages(prevMessages => {
        console.log('Adding project creation message to chat');
        return [...prevMessages, { text: 'We are creating your project structure...', isUser: false, step: currentStep }];
      });
    } else if (isConfirmation && message === 'NO') {
      console.log('User declined project creation');
      setCreatingProject(false);
      console.log('creatingProject set to false');
    }

    // Prepara o payload para enviar ao WebSocket
    const payload = {
      conversation_id: conversationID,
      message: message,
      is_confirmation: isConfirmation
    };

    console.log('Sending payload to WebSocket:', payload);

    // Envia a mensagem pelo WebSocket
    socketRef.current.send(JSON.stringify(payload));
    console.log('Message sent to WebSocket');

    // Note que não estamos definindo isLoading como false aqui.
    // Isso será feito quando recebermos uma resposta do servidor.
  };

  const handleConfirmation = (answer) => {
    sendMessage(answer, true);
    if (answer === "YES") {
      setProjectName("chat-app-maker");
    }
  };

  const toggleDarkMode = () => {
    setIsDarkMode(!isDarkMode);
    document.documentElement.classList.toggle('dark');
  };

  const handleFileSelect = (filePath) => {
    console.log(`File selected: ${filePath}`);

    setSelectedFile(filePath);

    const projectName = "chat-app-maker";
    const url = `http://localhost:8080/readFile?path=${encodeURIComponent(filePath)}&project=${encodeURIComponent(projectName)}`;

    console.log(`Requesting file content from: ${url}`);

    fetch(url)
      .then((response) => {
        console.log(`Response status: ${response.status}`);
        if (!response.ok) {
          return response.text().then(text => {
            throw new Error(`HTTP error! status: ${response.status}, message: ${text}`);
          });
        }
        return response.json();
      })
      .then((data) => {
        console.log(`Received data for file: ${filePath}`);
        if (data && data.content) {
          setFileContents((prevContents) => ({
            ...prevContents,
            [filePath]: data.content,
          }));
        } else {
          throw new Error('No content available in the response');
        }
      })
      .catch((error) => {
        console.error(`Error fetching file content for ${filePath}:`, error);
        setFileContents((prevContents) => ({
          ...prevContents,
          [filePath]: `Error fetching file content: ${error.message}`,
        }));
      });

    setShowChat(false);
  };

  return (
    <div className={`flex flex-col h-screen ${isDarkMode ? 'dark' : ''} bg-background`}>
      <div className="bg-white dark:bg-gray-800 shadow-md p-4 flex justify-between items-center">
        <div className="flex items-center space-x-6">
          <div className="text-xl font-bold text-gray-800 dark:text-white">Chat App</div>
          <nav className="flex space-x-4">
            <Button variant="ghost" size="sm">
              <Home className="h-5 w-5 mr-2" />
              Home
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setShowChat(!showChat)}>
              <MessageSquare className="h-5 w-5 mr-2" />
              {showChat ? 'Hide Chat' : 'Show Chat'}
            </Button>
            <Button variant="ghost" size="sm">
              <HelpCircle className="h-5 w-5 mr-2" />
              Help
            </Button>
          </nav>
        </div>
        <div className="flex space-x-4">
          <Button variant="ghost" size="icon" onClick={toggleDarkMode}>
            {isDarkMode ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
          </Button>
          <Button variant="ghost" size="icon">
            <User className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon">
            <Settings className="h-5 w-5" />
          </Button>
        </div>
      </div>

      <div className="flex-grow overflow-hidden flex p-4">
        <Card className="w-1/3 mr-4 bg-white dark:bg-gray-800 overflow-y-auto">
          <CardContent>
            <h3 className="text-lg font-semibold mb-2">Project Structure</h3>
            {projectStructure && (
              <FileTree structure={projectStructure} onFileSelect={handleFileSelect} />
            )}
          </CardContent>
        </Card>
        <Card className="w-2/3 flex flex-col bg-white dark:bg-gray-800">
          <CardContent className="flex-grow overflow-y-auto p-4">
            {selectedFile && !showChat ? (
              <div>
                <h3 className="text-lg font-semibold mb-2">{selectedFile}</h3>
                <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded-lg overflow-x-auto">
                  <code>{fileContents[selectedFile] || 'File content not available'}</code>
                </pre>
              </div>
            ) : showChat ? (
              <>
                <div className="h-8"></div>
                {messages.map((msg, index) => (
                  <ChatMessage
                    key={index}
                    message={msg.text}
                    isUser={msg.isUser}
                    step={msg.step}
                    isDownloadButton={msg.isDownloadButton}
                    projectName={projectName}
                  />
                ))}
                {isLoading && (
                  <div className="flex justify-center items-center my-4">
                    <Loader className="animate-spin h-6 w-6 text-blue-500" />
                  </div>
                )}
                {creatingProject && (
                  <div className="mt-4">
                    <ProgressBar percentage={progress.percentage} message={progress.message} />
                  </div>
                )}
                <div ref={messagesEndRef} />
              </>
            ) : (
              <div className="flex justify-center items-center h-full">
                <p>Select a file to view its content or click "Show Chat" to return to the conversation.</p>
              </div>
            )}
          </CardContent>

          {showChat && (
            <div className="p-4 border-t dark:border-gray-700">
              {awaitingConfirmation ? (
                <div className="flex space-x-2">
                  <Button onClick={() => handleConfirmation("YES")} className="flex-1 bg-green-500 hover:bg-green-600">
                    YES
                  </Button>
                  <Button onClick={() => handleConfirmation("NO")} className="flex-1 bg-red-500 hover:bg-red-600">
                    NO
                  </Button>
                </div>
              ) : (
                <div className="flex space-x-2">
                  <Select value={inputType} onValueChange={setInputType}>
                    <SelectTrigger className="w-[180px]">
                      <SelectValue placeholder="Select input type" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="text">Text</SelectItem>
                      <SelectItem value="yesno">Yes/No</SelectItem>
                    </SelectContent>
                  </Select>
                  {inputType === 'text' ? (
                    <Input
                      type="text"
                      placeholder="Digite sua mensagem..."
                      value={inputMessage}
                      onChange={(e) => setInputMessage(e.target.value)}
                      onKeyPress={(e) => e.key === 'Enter' && sendMessage(inputMessage)}
                      className="flex-grow dark:bg-gray-700 dark:text-white"
                    />
                  ) : (
                    <Select value={inputMessage} onValueChange={setInputMessage}>
                      <SelectTrigger className="flex-grow">
                        <SelectValue placeholder="Select Yes or No" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="Yes">Yes</SelectItem>
                        <SelectItem value="No">No</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                  <Button onClick={() => sendMessage(inputMessage)} className="bg-blue-500 hover:bg-blue-600 dark:bg-blue-600 dark:hover:bg-blue-700">
                    <Send size={20} />
                  </Button>
                </div>
              )}
            </div>
          )}
        </Card>
      </div>
      <div className="p-2 text-sm text-gray-600 dark:text-gray-400">
        Connection Status: {connectionStatus}
        {connectionDetails && <p className="text-xs">{connectionDetails}</p>}
      </div>
    </div>
  );
};

export default ChatApp;