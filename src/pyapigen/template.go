package main

import (
	"text/template"
)

var Template = template.Must(template.New("py").Parse(`'''
Copyright (2017) Sandia Corporation.
Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
the U.S. Government retains certain rights in this software.

Devin Cook <devcook@sandia.gov>
Jon Crussell <jcrusse@sandia.gov>

minimega bindings for Python

**************************************************************************
* THIS FILE IS AUTOMATICALLY GENERATED. DO NOT MODIFY THIS FILE BY HAND. *
**************************************************************************

This API uses a UNIX domain socket to communicate with a running instance of
minimega. The protocol is documented here, under "Command Port and the Local
Command Flag":

	http://minimega.org/articles/usage.article#TOC_2.2.

This file is automatically generated from the output of "minimega -cli". See
"doc.bash" for details on how to regenerate this file using pyapigen.

This API *should* work for both python2.7 and python3. Please report any issues
to the bug tracker:

	https://github.com/sandia-minimega/minimega/issues
'''


import copy
import json
import socket
import threading


# This is the revision of minimega these bindings were created for and the date
# that the bindings were generated.
__version__ = '{{ .Version }}'
__date__ = '{{ .Date }}'


class Error(Exception): pass


# HAX: python 2/3 hack
try:
	basestring
	def _isstr(obj):
		return isinstance(obj, basestring)
except NameError:
	def _isstr(obj):
		return isinstance(obj, str)


def connect(path='/tmp/minimega/minimega', raise_errors=True, debug=False, namespace=None):
	'''
	Connect to the minimega instance with UNIX socket at <path> and return a
	new minimega API object. See help(minimega.minimega) for an explaination of
	the other parameters.
	'''
	mm = minimega(path, raise_errors, debug, namespace)
	for resp in mm.version():
		if __version__ not in resp['Response']:
			print('WARNING: API was built using a different version of minimega')
	return mm


def print_rows(resps):
	'''
	print_rows walks the response from minimega and prints all tabular data.
	'''
	for resp in resps:
		for row in resp['Tabular'] or []:
			print(row)


def discard(mm):
	'''
	discard streams responses from minimega until there are none left
	'''
	try:
		for _ in mm.streamResponses():
			pass
	except Exception as e:
		if str(e) != 'no responses to stream from last command':
			raise


class minimega(object):
	'''
	This class communicates with a running instance of minimega using a UNIX
	domain socket.

	Each minimega command can be called from this object, and the response will
	be returned unless an Exception is thrown.
	'''

	def __init__(self, path, raise_errors, debug, namespace):
		'''
		Connects to the minimega instance with UNIX socket at <path>. If
		<raise_errors> is set, the Python APIs will raise an Exception whenever
		minimega returns a response with an error. If <debug> is set, debugging
		information will be printed. The <namespace> parameter allows you to
		"bind" the minimega object to a particular namespace (see
		help(minimega.minimega.namespace) for more info on namespaces).
		'''

		self.moreResponses = False

		self._lock = threading.Lock()
		self._path = path
		self._raise_errors = raise_errors
		self._debug = debug
		self._namespace = namespace
		self._socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
		self._socket.connect(path)


	def _get_response(self):
		'''
		_get_response reads a single response from minimega
		'''

		msg = ''
		more = self._socket.recv(1).decode('utf-8')
		response = None
		while response is None and more:
			msg += more
			# want to read a full JSON object, and not run json.loads for
			# every byte read
			if more != '}':
				more = self._socket.recv(1).decode('utf-8')
				continue

			try:
				response = json.loads(msg)
			except ValueError as e:
				if self._debug:
					print(e)
			more = self._socket.recv(1).decode('utf-8')

		if not response:
			raise Error('Expected response, socket closed')

		if self._debug:
			print('[debug] response: ' + str(response))
		if self._raise_errors:
			for resp in response['Resp']:
				if resp['Error'] != '':
					raise Error(resp['Error'])

		return response


	def _run(self, *args):
		'''
		_run sends a command to minimega and returns the first response,
		setting a flag if there are more responses to read with
		streamResponses.
		'''
		cmd = list(args)
		if self._namespace is not None:
			cmd = ['namespace', self._namespace] + cmd

		msg = json.dumps({'Command': ' '.join([str(v) for v in cmd if v])})

		with self._lock:
			if self.moreResponses:
				raise Error('more responses to be read from last command')

			if self._debug:
				print('[debug] sending cmd: ' + msg)

			if len(msg) != self._socket.send(msg.encode('utf-8')):
				raise Error('failed to write message to minimega')

			response = self._get_response()
			if response['More']:
				self.moreResponses = True

			return response['Resp']

	def namespace(self, name):
		'''
		Returns a new instance for use in a with statement:

			with mm.namespace("name") as mm:
				mm.vm_info(...)

		Note that this wraps the existing minimega connection so multiple
		instances can be used concurrently but the commands themselves will be
		executed serially.
		'''

		class __namespacer__:
			def __enter__(_):
				# clone the current instance and update the namespace
				mm = copy.copy(self)
				mm._namespace = name

				return mm

			def __exit__(self, type, value, traceback):
				pass

		return __namespacer__()


	def streamResponses(self):
		'''
		streamResponses returns a generator for additional responses to a
		previous command.
		'''

		with self._lock:
			if not self.moreResponses:
				raise Error('no responses to stream from last command')

			self.moreResponses = False

			response = self._get_response()

			while response['More']:
				yield response['Resp']
				response = self._get_response()

			yield response['Resp']


	{{ range $cmd := .Commands }}
	def {{ $cmd.Name }}(self,
	{{- range $arg := $cmd.Args -}}
		{{ $arg.Name }}{{ if $arg.Optional }}=None{{ end }},
	{{- end }}):
		'''
Variants:
	{{- range $i, $v := $cmd.Variants }}
	{{ $v.Pattern }}
	{{- end }}

{{ $cmd.Help }}
		'''

	{{- range $i, $v := $cmd.Variants }}
		# {{ $v.Pattern }}
		if {{ range $j, $arg := $v.Args -}}
			{{ if (gt $j 0) }} and {{ end -}}
				{{ $arg.Name }} != None
		{{- else -}}
		True
		{{- end }}:

		{{- range $arg := $v.Args }}
			{{- if gt (len $arg.Options) 1 }}
			# Validate that choice was valid for {{ $arg.Name }}
			if {{ $arg.Name }} not in [{{ range $o := $arg.Options }}{{ printf "%q" $o}},{{end}}]:
				raise ValueError("invalid value for {{ $arg.Name }}")
				{{- end -}}
			{{- end }}
			return self._run({{ $v.Template }})
	{{ end }}

		# didn't match any variant
		raise ValueError("invalid argument combination")
	{{ end }}
`))
