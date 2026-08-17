import assert from 'node:assert/strict'
import test from 'node:test'

import { composeCreateMode, isValidComposeName } from './compose-form.ts'

test('compose name allows docker project characters', () => {
  assert.equal(isValidComposeName('blog'), true)
  assert.equal(isValidComposeName('Blog_1.test'), true)
  assert.equal(isValidComposeName(''), false)
  assert.equal(isValidComposeName('-bad'), false)
  assert.equal(isValidComposeName('has space'), false)
})

test('create mode is exclusive', () => {
  assert.equal(composeCreateMode({ content: 'services: {}', path: '' }), 'create')
  assert.equal(composeCreateMode({ content: '', path: '/opt/app/docker-compose.yml' }), 'attach')
  assert.equal(composeCreateMode({ content: 'x', path: '/opt/app/docker-compose.yml' }), 'invalid')
  assert.equal(composeCreateMode({ content: '', path: '' }), 'invalid')
})
