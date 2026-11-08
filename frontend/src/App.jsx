
import { useEffect, useState } from 'react'
import {
  MapContainer,
  TileLayer,
  Marker,
  Popup,
  useMapEvents,
} from 'react-leaflet'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import './App.css'

const username = 'alice'

// Map marker
const markerIcon = L.divIcon({
  className: 'planet-marker',
  html: '<div class="marker-glow"></div>',
  iconSize: [22, 22],
  iconAnchor: [11, 11],
})

// Flag marker for selecting a post location
const flagIcon = L.divIcon({
  className: 'placement-flag-marker',
  html: '<div class="placement-flag"></div>',
  iconSize: [22, 28],
  iconAnchor: [5, 26],
})

// Detect clicks on the map
function MapClickHandler({ onMapClick }) {
  useMapEvents({
    click(event) {
      onMapClick(event.latlng)
    },
  })

  return null
}

function App() {
  // Posts
  const [posts, setPosts] = useState([])
  const [selectedPost, setSelectedPost] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deletingPostID, setDeletingPostID] = useState(null)

  // Profile
  const [profile, setProfile] = useState(null)
  const [profileLoading, setProfileLoading] = useState(true)
  const [editingProfile, setEditingProfile] = useState(false)
  const [savingProfile, setSavingProfile] = useState(false)
  const [bio, setBio] = useState('')
  const [planetName, setPlanetName] = useState('')
  const [planetDescription, setPlanetDescription] = useState('')

  // New posts
  const [showPostForm, setShowPostForm] = useState(false)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [latitude, setLatitude] = useState(40.7128)
  const [longitude, setLongitude] = useState(-74.006)
  const [placementPosition, setPlacementPosition] = useState(null)

  // Comments
  const [comments, setComments] = useState([])
  const [newComment, setNewComment] = useState('')
  const [replyingTo, setReplyingTo] = useState(null)
  const [replyContent, setReplyContent] = useState('')

  // Global chat
  const [channels, setChannels] = useState([])
  const [selectedChannel, setSelectedChannel] = useState(null)
  const [chatMessages, setChatMessages] = useState([])
  const [chatLoading, setChatLoading] = useState(false)
  const [chatError, setChatError] = useState('')
  const [chatInput, setChatInput] = useState('')
  const [sendingMessage, setSendingMessage] = useState(false)

  // Retrieve all posts
  async function fetchPosts() {
    try {
      const response = await fetch('/posts')

      if (!response.ok) {
        throw new Error('Could not retrieve posts')
      }

      const data = await response.json()

      setPosts(data)
      setError('')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  // Retrieve the current user's profile
  async function fetchProfile() {
    try {
      const response = await fetch(
        `/users?username=${encodeURIComponent(username)}`
      )

      if (!response.ok) {
        throw new Error('Could not retrieve profile')
      }

      const data = await response.json()

      setProfile(data)
      setBio(data.bio || '')
      setPlanetName(data.planet_name || '')
      setPlanetDescription(data.planet_description || '')
    } catch (err) {
      setError(err.message)
    } finally {
      setProfileLoading(false)
    }
  }

  // Retrieve comments for a post
  async function fetchComments(postID) {
    try {
      const response = await fetch(`/comments?post_id=${postID}`)

      if (!response.ok) {
        throw new Error('Could not retrieve comments')
      }

      const data = await response.json()

      setComments(data)
    } catch (err) {
      setError(err.message)
      setComments([])
    }
  }

  // Retrieve available chat channels
  async function fetchChannels() {
    try {
      const response = await fetch('/chat/channels')

      if (!response.ok) {
        throw new Error('Could not retrieve chat channels')
      }

      const data = await response.json()

      setChannels(data)

      if (data.length > 0) {
        if (selectedChannel === null) {
          setSelectedChannel(data[0])
        }
      }

      setChatError('')
    } catch (err) {
      setChatError(err.message)
    }
  }

  // Retrieve messages from a chat channel
  async function fetchChatMessages(channelID) {
    try {
      const response = await fetch(
        `/chat/messages?channel_id=${channelID}`
      )

      if (!response.ok) {
        throw new Error('Could not retrieve chat messages')
      }

      const data = await response.json()

      setChatMessages(data)
      setChatError('')
    } catch (err) {
      setChatError(err.message)
    } finally {
      setChatLoading(false)
    }
  }

  // Tell the server that this user is online
  async function sendHeartbeat() {
    try {
      const response = await fetch(
        `/users/heartbeat?username=${encodeURIComponent(username)}`,
        {
          method: 'POST',
        }
      )

      if (!response.ok) {
        throw new Error('Could not update online status')
      }
    } catch (err) {
      console.error('Heartbeat failed:', err)
    }
  }

  // Load the initial application data
  useEffect(() => {
    fetchPosts()
    fetchProfile()
    fetchChannels()
  }, [])

  // Send a heartbeat every 20 seconds
  useEffect(() => {
    sendHeartbeat()

    const interval = setInterval(function () {
      sendHeartbeat()
    }, 20000)

    return function () {
      clearInterval(interval)
    }
  }, [])

  // Refresh posts every 10 seconds
  useEffect(() => {
    const interval = setInterval(function () {
      fetchPosts()
    }, 10000)

    return function () {
      clearInterval(interval)
    }
  }, [])

  // Refresh chat messages every 4 seconds
  useEffect(() => {
    if (selectedChannel === null) {
      return
    }

    setChatMessages([])
    setChatLoading(true)

    fetchChatMessages(selectedChannel.id)

    const interval = setInterval(function () {
      fetchChatMessages(selectedChannel.id)
    }, 4000)

    return function () {
      clearInterval(interval)
    }
  }, [selectedChannel])

  // Load comments when a post is selected
  useEffect(() => {
    if (selectedPost !== null) {
      setComments([])
      fetchComments(selectedPost.id)
    } else {
      setComments([])
    }
  }, [selectedPost])

  // Save profile changes
  async function saveProfile(event) {
    event.preventDefault()

    setSavingProfile(true)
    setError('')

    try {
      const response = await fetch('/users', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: username,
          bio: bio,
          planet_name: planetName,
          planet_description: planetDescription,
        }),
      })

      if (!response.ok) {
        throw new Error(await response.text())
      }

      const updatedProfile = await response.json()

      setProfile(updatedProfile)
      setBio(updatedProfile.bio || '')
      setPlanetName(updatedProfile.planet_name || '')
      setPlanetDescription(updatedProfile.planet_description || '')
      setEditingProfile(false)
    } catch (err) {
      setError(err.message)
    } finally {
      setSavingProfile(false)
    }
  }

  // Cancel profile editing
  function cancelProfileEdit() {
    if (profile !== null) {
      setBio(profile.bio || '')
      setPlanetName(profile.planet_name || '')
      setPlanetDescription(profile.planet_description || '')
    }

    setEditingProfile(false)
  }

  // Create a new post
  async function createPost(event) {
    event.preventDefault()

    try {
      const response = await fetch('/posts', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          username: username,
          title: title,
          content: content,
          latitude: Number(latitude),
          longitude: Number(longitude),
        }),
      })

      if (!response.ok) {
        throw new Error(await response.text())
      }

      setTitle('')
      setContent('')
      setShowPostForm(false)
      setPlacementPosition(null)

      await fetchPosts()

      setError('')
    } catch (err) {
      setError(err.message)
    }
  }

  // Delete a post
  async function deletePost(post) {
    const confirmed = window.confirm(
      `Are you sure you want to delete "${post.title}"? This will also delete its comments and replies.`
    )

    if (!confirmed) {
      return
    }

    setDeletingPostID(post.id)
    setError('')

    try {
      const response = await fetch(`/posts?id=${post.id}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error(await response.text())
      }

      const updatedPosts = posts.filter(function (item) {
        return item.id !== post.id
      })

      setPosts(updatedPosts)

      if (selectedPost !== null && selectedPost.id === post.id) {
        setSelectedPost(null)
        setComments([])
      }

      setError('')
    } catch (err) {
      setError(err.message)
    } finally {
      setDeletingPostID(null)
    }
  }

  // Create a comment or reply
  async function createComment(event, parentCommentID = null) {
    event.preventDefault()

    let commentContent = newComment

    if (parentCommentID !== null) {
      commentContent = replyContent
    }

    if (commentContent.trim() === '' || selectedPost === null) {
      return
    }

    try {
      const response = await fetch(
        `/comments?post_id=${selectedPost.id}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            username: username,
            content: commentContent,
            parent_comment_id: parentCommentID,
          }),
        }
      )

      if (!response.ok) {
        throw new Error(await response.text())
      }

      setNewComment('')
      setReplyContent('')
      setReplyingTo(null)
      setError('')

      await fetchComments(selectedPost.id)
    } catch (err) {
      setError(err.message)
    }
  }

  // Send a message in global chat
  async function sendChatMessage(event) {
    event.preventDefault()

    const message = chatInput.trim()

    if (message === '' || selectedChannel === null || sendingMessage) {
      return
    }

    setSendingMessage(true)
    setChatError('')

    try {
      const response = await fetch(
        `/chat/messages?channel_id=${selectedChannel.id}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            username: username,
            content: message,
          }),
        }
      )

      if (!response.ok) {
        throw new Error(await response.text())
      }

      setChatInput('')

      await fetchChatMessages(selectedChannel.id)
    } catch (err) {
      setChatError(err.message)
    } finally {
      setSendingMessage(false)
    }
  }

  // Display comments and their replies
  function renderComments(parentID = null) {
    const result = []

    for (let i = 0; i < comments.length; i++) {
      const comment = comments[i]

      if (comment.parent_comment_id === parentID) {
        result.push(
          <div className="comment" key={comment.id}>
            <div className="comment-heading">
              <span className="comment-avatar">
                {comment.author.charAt(0).toUpperCase()}
              </span>

              <strong>{comment.author}</strong>
            </div>

            <p>{comment.content}</p>

            <button
              className="reply-button"
              onClick={function () {
                setReplyingTo(comment.id)
                setReplyContent('')
              }}
            >
              Reply
            </button>

            {replyingTo === comment.id && (
              <form
                className="reply-form"
                onSubmit={function (event) {
                  createComment(event, comment.id)
                }}
              >
                <input
                  value={replyContent}
                  onChange={function (event) {
                    setReplyContent(event.target.value)
                  }}
                  placeholder="Write a reply..."
                  required
                />

                <button type="submit">Send</button>
              </form>
            )}

            <div className="nested-comments">
              {renderComments(comment.id)}
            </div>
          </div>
        )
      }
    }

    return result
  }

  // Handle clicks on the map
  function handleMapClick(position) {
    setLatitude(position.lat.toFixed(6))
    setLongitude(position.lng.toFixed(6))
    setPlacementPosition([position.lat, position.lng])
    setShowPostForm(true)
  }

  // Format chat message timestamps
  function formatMessageTime(timestamp) {
    return new Date(timestamp).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <span className="brand-icon">✦</span>
          <span>PLANETARY</span>
        </div>

        <div className="user-info">
          <span className="status-dot online"></span>
          {username}
        </div>
      </header>

      <main className="main-layout">
        <aside className="sidebar">
          <div className="sidebar-heading">
            <div>
              <span className="eyebrow">YOUR UNIVERSE</span>
              <h1>Explore</h1>
            </div>

            <button
              className="new-post-button"
              onClick={function () {
                setShowPostForm(!showPostForm)
              }}
            >
              + Post
            </button>
          </div>

          {error !== '' && (
            <div className="error-message">{error}</div>
          )}

          {/* Profile */}
          <section className="profile-card">
            <div className="profile-card-heading">
              <span className="eyebrow">YOUR PROFILE</span>

              {!editingProfile && !profileLoading && (
                <button
                  className="edit-profile-button"
                  onClick={function () {
                    setEditingProfile(true)
                  }}
                >
                  Edit Profile
                </button>
              )}
            </div>

            {profileLoading ? (
              <p className="empty-message">Loading profile...</p>
            ) : editingProfile ? (
              <form className="profile-form" onSubmit={saveProfile}>
                <label>
                  Username
                  <input value={username} disabled />
                </label>

                <label>
                  Home planet
                  <input
                    value={planetName}
                    onChange={function (event) {
                      setPlanetName(event.target.value)
                    }}
                    placeholder="Give your planet a name"
                    maxLength={60}
                    required
                  />
                </label>

                <label>
                  Your bio
                  <textarea
                    value={bio}
                    onChange={function (event) {
                      setBio(event.target.value)
                    }}
                    placeholder="Tell the universe about yourself..."
                    maxLength={300}
                    rows={3}
                  />
                </label>

                <label>
                  Planet description
                  <textarea
                    value={planetDescription}
                    onChange={function (event) {
                      setPlanetDescription(event.target.value)
                    }}
                    placeholder="Describe your home planet..."
                    maxLength={500}
                    rows={4}
                  />
                </label>

                <div className="profile-form-actions">
                  <button
                    className="submit-button"
                    type="submit"
                    disabled={savingProfile}
                  >
                    {savingProfile ? 'Saving...' : 'Save profile'}
                  </button>

                  <button
                    className="cancel-profile-button"
                    type="button"
                    onClick={cancelProfileEdit}
                    disabled={savingProfile}
                  >
                    Cancel
                  </button>
                </div>
              </form>
            ) : profile !== null ? (
              <div className="profile-content">
                <div className="profile-avatar">
                  {profile.username.charAt(0).toUpperCase()}
                </div>

                <h2>{profile.username}</h2>

                <p className="profile-bio">
                  {profile.bio || 'No bio yet. Tell the universe about yourself!'}
                </p>

                <div className="planet-profile">
                  <div className="planet-symbol">◉</div>

                  <div>
                    <span className="planet-label">HOME PLANET</span>
                    <h3>{profile.planet_name}</h3>
                  </div>
                </div>

                <p className="planet-description">
                  {profile.planet_description ||
                    'This planet is waiting to be explored.'}
                </p>
              </div>
            ) : (
              <p className="empty-message">Profile could not be loaded.</p>
            )}
          </section>

          {/* Global chat */}
          <section className="chat-panel">
            <div className="chat-panel-heading">
              <div>
                <span className="eyebrow">COMMUNITY</span>
                <h2>Global Chat</h2>
              </div>

              <span className="chat-online-indicator">
                <span className="online-dot"></span>
                GLOBAL
              </span>
            </div>

            <div className="chat-channel-list">
              {channels.map(function (channel) {
                let activeClass = ''

                if (
                  selectedChannel !== null &&
                  selectedChannel.id === channel.id
                ) {
                  activeClass = 'active'
                }

                return (
                  <button
                    key={channel.id}
                    className={`chat-channel ${activeClass}`}
                    onClick={function () {
                      setSelectedChannel(channel)
                    }}
                  >
                    <span># {channel.name}</span>
                  </button>
                )
              })}

              {channels.length === 0 && (
                <p className="empty-message">
                  {chatError || 'Loading channels...'}
                </p>
              )}
            </div>

            {selectedChannel !== null && (
              <>
                <div className="chat-channel-description">
                  {selectedChannel.description}
                </div>

                <div className="chat-message-list">
                  {chatLoading && chatMessages.length === 0 ? (
                    <p className="empty-message">Loading messages...</p>
                  ) : chatMessages.length === 0 ? (
                    <p className="empty-message">
                      No messages yet. Start the conversation!
                    </p>
                  ) : (
                    chatMessages.map(function (message) {
                      let statusClass = 'offline'
                      let statusTitle = 'Offline'

                      if (message.online) {
                        statusClass = 'online'
                        statusTitle = 'Online'
                      }

                      return (
                        <div className="chat-message" key={message.id}>
                          <div className="chat-message-heading">
                            <span className="chat-avatar">
                              {message.author.charAt(0).toUpperCase()}
                            </span>

                            <strong>{message.author}</strong>

                            <span
                              className={`status-dot ${statusClass}`}
                              title={statusTitle}
                            ></span>

                            <time>
                              {formatMessageTime(message.created_at)}
                            </time>
                          </div>

                          <p>{message.content}</p>
                        </div>
                      )
                    })
                  )}
                </div>

                <form className="chat-form" onSubmit={sendChatMessage}>
                  <input
                    value={chatInput}
                    onChange={function (event) {
                      setChatInput(event.target.value)
                    }}
                    placeholder={`Message #${selectedChannel.name}`}
                    maxLength={1000}
                    disabled={sendingMessage}
                  />

                  <button
                    type="submit"
                    disabled={chatInput.trim() === '' || sendingMessage}
                  >
                    {sendingMessage ? '...' : 'Send'}
                  </button>
                </form>

                {chatError !== '' && (
                  <p className="chat-error">{chatError}</p>
                )}
              </>
            )}
          </section>

          {/* New post form */}
          {showPostForm && (
            <form className="post-form" onSubmit={createPost}>
              <h2>New transmission</h2>

              <input
                value={title}
                onChange={function (event) {
                  setTitle(event.target.value)
                }}
                placeholder="Post title"
                required
              />

              <textarea
                value={content}
                onChange={function (event) {
                  setContent(event.target.value)
                }}
                placeholder="Share something with the universe..."
                required
              />

              <div className="coordinate-inputs">
                <label>
                  Latitude
                  <input
                    type="number"
                    step="any"
                    min="-90"
                    max="90"
                    value={latitude}
                    onChange={function (event) {
                      setLatitude(event.target.value)
                    }}
                    required
                  />
                </label>

                <label>
                  Longitude
                  <input
                    type="number"
                    step="any"
                    min="-180"
                    max="180"
                    value={longitude}
                    onChange={function (event) {
                      setLongitude(event.target.value)
                    }}
                    required
                  />
                </label>
              </div>

              <p className="form-hint">
                You can also click anywhere on the map to select a location.
              </p>

              <button className="submit-button" type="submit">
                Publish post
              </button>
            </form>
          )}

          {/* Recent posts */}
          <div className="feed-heading">
            <h2>Recent transmissions</h2>
            <span>{posts.length} posts</span>
          </div>

          <div className="post-list">
            {loading ? (
              <p className="empty-message">Loading transmissions...</p>
            ) : posts.length === 0 ? (
              <p className="empty-message">No posts yet. Start exploring!</p>
            ) : (
              posts.map(function (post) {
                let statusClass = 'offline'
                let statusTitle = 'Offline'

                if (post.online) {
                  statusClass = 'online'
                  statusTitle = 'Online'
                }

                let activeClass = ''

                if (
                  selectedPost !== null &&
                  selectedPost.id === post.id
                ) {
                  activeClass = 'active'
                }

                return (
                  <div
                    className={`post-card ${activeClass}`}
                    key={post.id}
                    onClick={function () {
                      setSelectedPost(post)
                    }}
                    role="button"
                    tabIndex={0}
                    onKeyDown={function (event) {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault()
                        setSelectedPost(post)
                      }
                    }}
                  >
                    <div className="post-card-top">
                      <span className="post-author">
                        <span className="small-avatar">
                          {post.author.charAt(0).toUpperCase()}
                        </span>

                        {post.author}

                        <span
                          className={`status-dot ${statusClass}`}
                          title={statusTitle}
                        ></span>
                      </span>

                      <div className="post-card-actions">
                        <span className="post-coordinate">
                          #{post.id}
                        </span>

                        {post.author === username && (
                          <button
                            className="delete-post-button"
                            type="button"
                            disabled={deletingPostID === post.id}
                            onClick={function (event) {
                              event.stopPropagation()
                              deletePost(post)
                            }}
                          >
                            {deletingPostID === post.id
                              ? 'Deleting...'
                              : 'Delete'}
                          </button>
                        )}
                      </div>
                    </div>

                    <h3>{post.title}</h3>
                    <p>{post.content}</p>

                    <span className="location-text">
                      {Number(post.latitude).toFixed(3)},
                      {' '}
                      {Number(post.longitude).toFixed(3)}
                    </span>
                  </div>
                )
              })
            )}
          </div>

          {/* Selected post and comments */}
          {selectedPost !== null && (
            <section className="selected-post">
              <div className="detail-heading">
                <span className="eyebrow">TRANSMISSION DETAILS</span>

                <button
                  className="close-button"
                  onClick={function () {
                    setSelectedPost(null)
                  }}
                >
                  ×
                </button>
              </div>

              <h2>{selectedPost.title}</h2>

              <p className="detail-author">
                Posted by {selectedPost.author}
              </p>

              <p className="detail-content">
                {selectedPost.content}
              </p>

              {selectedPost.author === username && (
                <button
                  className="delete-post-button detail-delete-button"
                  disabled={deletingPostID === selectedPost.id}
                  onClick={function () {
                    deletePost(selectedPost)
                  }}
                >
                  {deletingPostID === selectedPost.id
                    ? 'Deleting...'
                    : 'Delete post'}
                </button>
              )}

              <div className="comments-heading">
                <h3>Comments</h3>
                <span>{comments.length}</span>
              </div>

              <form
                className="comment-form"
                onSubmit={function (event) {
                  createComment(event)
                }}
              >
                <input
                  value={newComment}
                  onChange={function (event) {
                    setNewComment(event.target.value)
                  }}
                  placeholder="Write a comment..."
                  required
                />

                <button type="submit">Send</button>
              </form>

              <div className="comment-list">
                {comments.length === 0 ? (
                  <p className="empty-message">No comments yet.</p>
                ) : (
                  renderComments()
                )}
              </div>
            </section>
          )}
        </aside>

        {/* Map */}
        <section className="map-section">
          <div className="map-overlay">
            <span className="map-status">
              <span className="online-dot"></span>
              LIVE MAP
            </span>

            <p>Click anywhere to create a transmission</p>
          </div>

          <MapContainer
            center={[40.7128, -74.006]}
            zoom={3}
            minZoom={2}
            maxZoom={18}
            worldCopyJump={true}
            className="planetary-map"
          >
            <TileLayer
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            />

            <MapClickHandler onMapClick={handleMapClick} />

            {placementPosition !== null && (
              <Marker
                position={placementPosition}
                icon={flagIcon}
                interactive={false}
              />
            )}

            {posts.map(function (post) {
              return (
                <Marker
                  key={post.id}
                  position={[post.latitude, post.longitude]}
                  icon={markerIcon}
                  eventHandlers={{
                    click: function () {
                      setSelectedPost(post)
                    },
                  }}
                >
                  <Popup>
                    <strong>{post.title}</strong>
                    <br />
                    By {post.author}
                  </Popup>
                </Marker>
              )
            })}
          </MapContainer>
        </section>
      </main>
    </div>
  )
}

export default App
