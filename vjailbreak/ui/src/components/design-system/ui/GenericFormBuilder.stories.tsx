import type { Meta, StoryObj } from '@storybook/react'
import { fn } from '@storybook/test'
import GenericFormBuilder from './GenericFormBuilder'

const meta: Meta<typeof GenericFormBuilder> = {
  title: 'Components/Design System/GenericFormBuilder',
  component: GenericFormBuilder,
  parameters: {
    layout: 'padded'
  },
  tags: ['autodocs'],
  argTypes: {
    onSubmit: { action: 'submitted' },
    onCancel: { action: 'cancelled' }
  }
}

export default meta
type Story = StoryObj<typeof meta>

// Basic form with common field types
export const Basic: Story = {
  args: {
    onSubmit: fn(),
    sections: [
      {
        id: 'basic-info',
        title: 'Basic Information',
        description: 'Enter your basic personal details',
        fields: [
          {
            name: 'firstName',
            label: 'First Name',
            type: 'text',
            placeholder: 'Enter your first name',
            rules: { required: 'First name is required' }
          },
          {
            name: 'lastName',
            label: 'Last Name',
            type: 'text',
            placeholder: 'Enter your last name',
            rules: { required: 'Last name is required' }
          },
          {
            name: 'email',
            label: 'Email Address',
            type: 'text',
            placeholder: 'Enter your email',
            rules: {
              required: 'Email is required',
              pattern: {
                value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                message: 'Invalid email address'
              }
            }
          },
          {
            name: 'age',
            label: 'Age',
            type: 'number',
            placeholder: 'Enter your age',
            rules: {
              required: 'Age is required',
              min: { value: 18, message: 'Must be at least 18' },
              max: { value: 120, message: 'Must be less than 120' }
            }
          }
        ]
      },
      {
        id: 'preferences',
        title: 'Preferences',
        fields: [
          {
            name: 'newsletter',
            label: 'Subscribe to newsletter',
            type: 'checkbox',
            description: 'Receive updates about new features'
          },
          {
            name: 'theme',
            label: 'Theme Preference',
            type: 'select',
            options: [
              { label: 'Light', value: 'light' },
              { label: 'Dark', value: 'dark' },
              { label: 'Auto', value: 'auto' }
            ],
            defaultValue: 'auto'
          },
          {
            name: 'notifications',
            label: 'Enable Notifications',
            type: 'switch',
            description: 'Get notified about important events'
          }
        ]
      }
    ],
    defaultValues: {
      firstName: '',
      lastName: '',
      email: '',
      age: '',
      newsletter: false,
      theme: 'auto',
      notifications: true
    }
  }
}

// Form with conditional visibility
export const ConditionalFields: Story = {
  args: {
    onSubmit: fn(),
    sections: [
      {
        id: 'account',
        title: 'Account Setup',
        fields: [
          {
            name: 'accountType',
            label: 'Account Type',
            type: 'select',
            options: [
              { label: 'Personal', value: 'personal' },
              { label: 'Business', value: 'business' }
            ],
            rules: { required: 'Please select an account type' }
          },
          {
            name: 'companyName',
            label: 'Company Name',
            type: 'text',
            placeholder: 'Enter company name',
            visible: { field: 'accountType', equals: 'business' },
            rules: {
              required: 'Company name is required for business accounts'
            }
          },
          {
            name: 'companySize',
            label: 'Company Size',
            type: 'select',
            options: [
              { label: '1-10', value: 'small' },
              { label: '11-50', value: 'medium' },
              { label: '51+', value: 'large' }
            ],
            visible: { field: 'accountType', equals: 'business' }
          }
        ]
      }
    ]
  }
}

// Form with array fields
export const ArrayFields: Story = {
  args: {
    onSubmit: fn(),
    sections: [
      {
        id: 'team',
        title: 'Team Members',
        description: 'Add team members to your project',
        fields: [
          {
            name: 'teamMembers',
            type: 'array',
            label: 'Team Members',
            minItems: 1,
            maxItems: 5,
            addLabel: 'Add Team Member',
            children: [
              {
                name: 'name',
                label: 'Name',
                type: 'text',
                placeholder: 'Member name',
                rules: { required: 'Name is required' }
              },
              {
                name: 'role',
                label: 'Role',
                type: 'select',
                options: [
                  { label: 'Developer', value: 'developer' },
                  { label: 'Designer', value: 'designer' },
                  { label: 'Manager', value: 'manager' }
                ],
                rules: { required: 'Role is required' }
              },
              {
                name: 'email',
                label: 'Email',
                type: 'text',
                placeholder: 'member@example.com',
                rules: {
                  required: 'Email is required',
                  pattern: {
                    value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                    message: 'Invalid email address'
                  }
                }
              }
            ]
          }
        ]
      }
    ],
    defaultValues: {
      teamMembers: [{ name: '', role: 'developer', email: '' }]
    }
  }
}

// Form with advanced field types
export const AdvancedFields: Story = {
  args: {
    onSubmit: fn(),
    sections: [
      {
        id: 'advanced',
        title: 'Advanced Fields',
        fields: [
          {
            name: 'bio',
            label: 'Biography',
            type: 'textarea',
            placeholder: 'Tell us about yourself...',
            description: 'Maximum 500 characters'
          },
          {
            name: 'birthDate',
            label: 'Date of Birth',
            type: 'date',
            description: 'Select your birth date'
          },
          {
            name: 'avatar',
            label: 'Profile Picture',
            type: 'file',
            description: 'Upload a profile picture (JPG, PNG)',
            accept: 'image/jpeg,image/png'
          },
          {
            name: 'contactMethod',
            label: 'Preferred Contact Method',
            type: 'radio',
            options: [
              { label: 'Email', value: 'email' },
              { label: 'Phone', value: 'phone' },
              { label: 'SMS', value: 'sms' }
            ]
          }
        ]
      }
    ]
  }
}

// Form with custom field renderer
export const CustomFields: Story = {
  args: {
    onSubmit: fn(),
    sections: [
      {
        id: 'custom',
        title: 'Custom Fields',
        fields: [
          {
            name: 'rating',
            label: 'Rating',
            type: 'custom',
            render: ({ value, onChange }) => (
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <span>Rate this:</span>
                {[1, 2, 3, 4, 5].map((star) => (
                  <button
                    key={star}
                    type="button"
                    onClick={() => onChange(star)}
                    style={{
                      fontSize: '20px',
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      color: star <= value ? '#gold' : '#ccc'
                    }}
                  >
                    {'★'}
                  </button>
                ))}
              </div>
            )
          },
          {
            name: 'color',
            label: 'Favorite Color',
            type: 'custom',
            render: ({ value, onChange }) => (
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <span>Choose a color:</span>
                <input
                  type="color"
                  value={value || '#000000'}
                  onChange={(e) => onChange(e.target.value)}
                  style={{ width: '50px', height: '30px' }}
                />
                <span>{value}</span>
              </div>
            )
          }
        ]
      }
    ]
  }
}

// Form with async options loading
export const AsyncOptions: Story = {
  args: {
    onSubmit: fn(),
    sections: [
      {
        id: 'async',
        title: 'Async Options',
        description: 'Demonstrates async options loading',
        fields: [
          {
            name: 'country',
            label: 'Country',
            type: 'select',
            loadOptions: async () => {
              // Simulate API call
              await new Promise((resolve) => setTimeout(resolve, 1000))
              return [
                { label: 'United States', value: 'us' },
                { label: 'Canada', value: 'ca' },
                { label: 'United Kingdom', value: 'uk' },
                { label: 'Germany', value: 'de' },
                { label: 'France', value: 'fr' }
              ]
            }
          },
          {
            name: 'dynamicOptions',
            label: 'Dynamic Options Based on Country',
            type: 'select',
            options: (values) => {
              if (values?.country === 'us') {
                return [
                  { label: 'California', value: 'ca' },
                  { label: 'New York', value: 'ny' },
                  { label: 'Texas', value: 'tx' }
                ]
              } else if (values?.country === 'ca') {
                return [
                  { label: 'Ontario', value: 'on' },
                  { label: 'Quebec', value: 'qc' },
                  { label: 'British Columbia', value: 'bc' }
                ]
              }
              return []
            },
            visible: { field: 'country' }
          }
        ]
      }
    ]
  }
}

// Form with validation errors display
export const WithValidation: Story = {
  args: {
    onSubmit: fn(),
    showErrors: true,
    sections: [
      {
        id: 'validation',
        title: 'Validation Example',
        description: 'This form demonstrates various validation rules',
        fields: [
          {
            name: 'requiredField',
            label: 'Required Field',
            type: 'text',
            placeholder: 'This field is required',
            rules: { required: 'This field cannot be empty' }
          },
          {
            name: 'emailField',
            label: 'Email Field',
            type: 'text',
            placeholder: 'Enter valid email',
            rules: {
              required: 'Email is required',
              pattern: {
                value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                message: 'Please enter a valid email address'
              }
            }
          },
          {
            name: 'minLength',
            label: 'Min Length (5 chars)',
            type: 'text',
            placeholder: 'At least 5 characters',
            rules: {
              required: 'This field is required',
              minLength: {
                value: 5,
                message: 'Must be at least 5 characters long'
              }
            }
          }
        ]
      }
    ]
  }
}

// Single column layout
export const SingleColumn: Story = {
  args: {
    ...Basic.args,
    columnCount: 1
  }
}

// Form with reset and cancel buttons
export const WithActions: Story = {
  args: {
    ...Basic.args,
    showReset: true,
    onCancel: fn(),
    submitLabel: 'Save Changes'
  }
}
