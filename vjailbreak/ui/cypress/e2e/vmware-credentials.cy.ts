describe('Add VMware Credentials', () => {
  const namespace = 'migration-system'

  beforeEach(() => {
    // Intercept initial credentials fetch - using wildcard to match potential /dev-api prefix
    cy.intercept('GET', `**/namespaces/${namespace}/vmwarecreds`, {
      body: { items: [] },
    }).as('getVmwareCreds')

    cy.intercept('GET', `**/namespaces/${namespace}/openstackcreds`, {
      body: { items: [] },
    }).as('getOpenstackCreds')

    cy.visit('/dashboard/credentials')
    cy.wait(['@getVmwareCreds', '@getOpenstackCreds'])
  })

  it('should successfully add VMware credentials', () => {
    const credName = 'test-vcenter-creds'
    const vCenterHost = 'vcenter.example.com'
    const datacenter = 'Datacenter-1'
    const username = 'admin@vsphere.local'
    const password = 'securepassword'

    // Open Drawer
    cy.contains('button', 'Add VMware Credentials').click()
    cy.get('[data-testid=vmware-cred-form]').should('be.visible')

    // Fill Form
    cy.get('input[name="credentialName"]').type(credName)
    cy.get('input[name="vcenterHost"]').type(vCenterHost)
    cy.get('input[name="datacenter"]').type(datacenter)
    cy.get('input[name="username"]').type(username)
    cy.get('input[name="password"]').type(password)
    
    // Test toggle
    cy.get('input[name="insecure"]').click({ force: true }) // Check if force needed for switch

    // Visual Snapshot of the filled form
    // Note: ensure consistent rendering (e.g. no animations in progress)
    //cy.matchImageSnapshot('vmware-credentials-drawer-filled')

    // Mock successful creation
    cy.intercept('POST', `**/namespaces/${namespace}/secrets`, {
      statusCode: 201,
      body: {
        metadata: {
          name: `${credName}-vmware-secret`,
          namespace,
        },
      },
    }).as('createSecret')

    cy.intercept('POST', `**/namespaces/${namespace}/vmwarecreds`, {
      statusCode: 201,
      body: {
        metadata: {
          name: credName,
          namespace,
        },
        spec: {
          secretRef: { name: `${credName}-vmware-secret` },
          datacenter: datacenter
        },
        status: {
            vmwareValidationStatus: 'Succeeded'
        }
      },
    }).as('createCreds')

    // Mock validation polling which happens after creation
    cy.intercept('GET', `**/namespaces/${namespace}/vmwarecreds/${credName}`, {
      body: {
        metadata: { name: credName, namespace },
        status: { vmwareValidationStatus: 'Succeeded' }
      }
    }).as('pollCreds')


    // Submit
    cy.get('[data-testid=vmware-cred-submit]').click()

    cy.wait('@createSecret').its('request.body.data').should((data) => {
        expect(data).to.have.property('VCENTER_PASSWORD')
    })
    
    cy.wait('@createCreds')
    
    // Should see success state in drawer (implied by pollCreds returning Succeeded)
    cy.contains('VMware credentials created').scrollIntoView().should('be.visible')
    
    // Wait for drawer to close (it closes after 1.5s on success)
    cy.get('[data-testid=vmware-cred-form]').should('not.exist')
  })

  it('should handle validation failure', () => {
    const credName = 'fail-creds'
    
    cy.contains('button', 'Add VMware Credentials').click()
    
    cy.get('input[name="credentialName"]').type(credName)
    cy.get('input[name="vcenterHost"]').type('fail.com')
    cy.get('input[name="datacenter"]').type('dc')
    cy.get('input[name="username"]').type('user')
    cy.get('input[name="password"]').type('pass')

    // Mock creation success but validation fail
    cy.intercept('POST', `**/namespaces/${namespace}/secrets`, { statusCode: 201, body: {} }).as('createSecret')
    cy.intercept('POST', `**/namespaces/${namespace}/vmwarecreds`, {
        statusCode: 201,
        body: { metadata: { name: credName, namespace } }
    }).as('createCreds')

    cy.intercept('GET', `**/namespaces/${namespace}/vmwarecreds/${credName}`, {
        body: {
          metadata: { name: credName, namespace },
          status: { vmwareValidationStatus: 'Failed', vmwareValidationMessage: 'Invalid credentials' }
        }
    }).as('pollCredsFail')

    cy.get('[data-testid=vmware-cred-submit]').click()
    
    cy.wait('@createSecret')
    cy.wait('@createCreds')
    cy.wait('@pollCredsFail') // might need to wait multiple times or just once depending on interval

    cy.contains('Invalid credentials').should('be.visible')
  })
})
